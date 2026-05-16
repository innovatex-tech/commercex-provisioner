package deploy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHOrchestrator struct {
	client *ssh.Client
	config *ssh.ClientConfig
	host   string
}

func NewSSHOrchestrator(host, user, password, keyPath string) (*SSHOrchestrator, error) {
	var auth []ssh.AuthMethod

	if password != "" {
		auth = append(auth, ssh.Password(password))
	}

	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if len(auth) == 0 {
		return nil, fmt.Errorf("no SSH authentication method provided")
	}

	// Add port if missing
	if !strings.Contains(host, ":") {
		host = host + ":22"
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For convenience, though production should use host keys
		Timeout:         10 * time.Second,
	}

	return &SSHOrchestrator{
		config: config,
		host:   host,
	}, nil
}

func (s *SSHOrchestrator) Connect() error {
	client, err := ssh.Dial("tcp", s.host, s.config)
	if err != nil {
		return err
	}
	s.client = client
	return nil
}

func (s *SSHOrchestrator) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// RunCommand executes a command on the remote server and returns output.
func (s *SSHOrchestrator) RunCommand(command string) (string, error) {
	if s.client == nil {
		if err := s.Connect(); err != nil {
			return "", err
		}
	}

	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b

	err = session.Run(command)
	return b.String(), err
}

// CheckPrerequisites verifies Docker and Docker Compose on remote.
func (s *SSHOrchestrator) CheckPrerequisites() error {
	out, err := s.RunCommand("docker --version")
	if err != nil {
		return fmt.Errorf("Docker not found on remote: %v (%s)", err, strings.TrimSpace(out))
	}

	out, err = s.RunCommand("docker compose version")
	if err != nil {
		// Try standalone docker-compose
		out2, err2 := s.RunCommand("docker-compose --version")
		if err2 != nil {
			return fmt.Errorf("Docker Compose v2 not found on remote: %v (%s)", err, strings.TrimSpace(out))
		}
		fmt.Printf("Warning: Remote uses docker-compose v1 (%s)\n", strings.TrimSpace(out2))
	}

	return nil
}

// PushFiles transfers a directory to the remote server using a tar pipe.
func (s *SSHOrchestrator) PushFiles(localDir, remotePath string) error {
	if s.client == nil {
		if err := s.Connect(); err != nil {
			return err
		}
	}

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 1. Ensure remote path exists
	_, err = s.RunCommand(fmt.Sprintf("mkdir -p '%s'", remotePath))
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// 2. Setup tar pipe
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	session.Stderr = &stderr

	err = session.Start(fmt.Sprintf("tar -xz -C '%s'", remotePath))
	if err != nil {
		return fmt.Errorf("failed to start remote tar: %v (%s)", err, stderr.String())
	}

	// 3. Create local tar stream
	if err := s.tarDir(localDir, stdin); err != nil {
		return fmt.Errorf("failed to tar local directory: %v", err)
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("remote tar failed: %v (%s)", err, stderr.String())
	}

	return nil
}

func (s *SSHOrchestrator) tarDir(src string, w io.WriteCloser) error {
	defer w.Close()
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Use relative path for header
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		return nil
	})
}

// DeployRemote runs docker compose up on the remote server.
func (s *SSHOrchestrator) DeployRemote(remotePath string) error {
	// Use 'docker compose' or fallback to 'docker-compose'
	cmd := fmt.Sprintf("cd '%s' && (docker compose up -d --build || docker-compose up -d --build)", remotePath)
	out, err := s.RunCommand(cmd)
	if err != nil {
		return fmt.Errorf("remote deploy failed: %v\nOutput: %s", err, out)
	}
	return nil
}
