package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds SSH connection parameters
type SSHConfig struct {
	Host    string
	User    string
	Port    string
	KeyPath string
}

// SSHClient manages SSH connections and file transfers
type SSHClient struct {
	config SSHConfig
	client *ssh.Client
}

// NewSSHClient creates a new SSH client (not yet connected)
func NewSSHClient(config SSHConfig) *SSHClient {
	if config.Port == "" {
		config.Port = "22"
	}
	return &SSHClient{config: config}
}

// parseDestination parses "user@host:/path" format
// returns (user, host, path, error)
func parseDestination(dest string) (user, host, dirPath string, err error) {
	// split by @
	parts := strings.Split(dest, "@")
	if len(parts) != 2 {
		err = fmt.Errorf("invalid format: expected user@host:/path")
		return
	}

	user = parts[0]
	hostAndPath := parts[1]

	// split by : to separate host from path
	parts = strings.SplitN(hostAndPath, ":", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("invalid format: expected user@host:/path")
		return
	}

	host = parts[0]
	dirPath = parts[1]
	return
}

// Connect establishes an SSH connection using key-based auth
func (sc *SSHClient) Connect() error {
	// determine key path
	keyPath := sc.config.KeyPath
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("no home directory: %w", err)
		}

		// list of common key locations to try
		possiblePaths := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
		}

		// find the first key that exists
		found := false
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				keyPath = path
				found = true
				break
			}
		}

		if !found {
			triedPaths := ""
			for _, p := range possiblePaths {
				triedPaths += "\n  - " + p
			}
			return fmt.Errorf(
				"no SSH private key found. tried:%s",
				triedPaths,
			)
		}
	}

	// read private key
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read private key at %s: %w", keyPath, err)
	}

	// parse the key
	key, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		// if it's an encrypted key error
		if err.Error() == "ssh: this private key is encrypted" {
			return fmt.Errorf(
				"SSH key is encrypted. "+
					"please generate an unencrypted key. "+
					"error: %w",
				err,
			)
		}
		return fmt.Errorf("parse private key from %s: %w", keyPath, err)
	}

	// create SSH config
	sshConfig := &ssh.ClientConfig{
		User: sc.config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// dial the remote host
	addr := fmt.Sprintf("%s:%s", sc.config.Host, sc.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf(
			"ssh dial %s (user: %s): %w",
			addr,
			sc.config.User,
			err,
		)
	}

	sc.client = client
	return nil
}

// Close closes the SSH connection
func (sc *SSHClient) Close() error {
	if sc.client != nil {
		return sc.client.Close()
	}
	return nil
}

// CreateDirectory creates a directory on the remote host
// equivalent to mkdir -p
func (sc *SSHClient) CreateDirectory(remotePath string) error {
	if sc.client == nil {
		return fmt.Errorf("not connected")
	}

	session, err := sc.client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf("mkdir -p %q", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

// TransferFile copies a single file to remote via SCP
// progressFn is called with (bytesTransferred, totalBytes) for progress tracking
func (sc *SSHClient) TransferFile(
	localPath,
	remotePath string,
	progressFn func(int64, int64),
) error {
	if sc.client == nil {
		return fmt.Errorf("not connected")
	}

	// open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	// get file size
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	totalSize := info.Size()

	// create session
	session, err := sc.client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// get stdin for scp
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	// start scp in sink (target) mode
	if err := session.Start(fmt.Sprintf("scp -t %q", remotePath)); err != nil {
		return fmt.Errorf("start scp: %w", err)
	}

	// send file metadata: C [permissions] [size] [filename]
	fmt.Fprintf(
		stdin,
		"C0644 %d %s\n",
		totalSize,
		path.Base(localPath),
	)

	// send file contents
	if _, err := io.CopyN(stdin, file, totalSize); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// update progress
	if progressFn != nil {
		progressFn(totalSize, totalSize)
	}

	// send final newline to mark end of file
	fmt.Fprint(stdin, "\n")
	stdin.Close()

	// wait for scp to finish
	if err := session.Wait(); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}

	return nil
}

// TransferFiles transfers multiple files to a remote directory
// sends progress updates via progressChan
// closes the channel when done
func (sc *SSHClient) TransferFiles(
	fileMap map[string]string, // local path -> remote filename
	remoteDir string,
	progressChan chan TransferProgressMessage,
) error {
	defer close(progressChan)

	// create remote directory first
	if err := sc.CreateDirectory(remoteDir); err != nil {
		progressChan <- TransferProgressMessage{
			FileName: "",
			Status:   "error",
			Error:    fmt.Sprintf("create remote dir: %v", err),
		}
		return err
	}

	// transfer each file
	for localPath, remoteFilename := range fileMap {
		remoteFullPath := path.Join(remoteDir, remoteFilename)

		// get file info
		info, err := os.Stat(localPath)
		if err != nil {
			progressChan <- TransferProgressMessage{
				FileName: localPath,
				Status:   "error",
				Error:    err.Error(),
			}
			continue
		}

		fileSize := info.Size()

		// send "starting" message
		progressChan <- TransferProgressMessage{
			FileName:         localPath,
			Status:           "transferring",
			BytesTransferred: 0,
			TotalBytes:       fileSize,
		}

		// do the transfer
		err = sc.TransferFile(
			localPath,
			remoteFullPath,
			func(transferred, total int64) {
				// progress callback
				progressChan <- TransferProgressMessage{
					FileName:         localPath,
					Status:           "transferring",
					BytesTransferred: transferred,
					TotalBytes:       total,
				}
			},
		)

		if err != nil {
			progressChan <- TransferProgressMessage{
				FileName: localPath,
				Status:   "error",
				Error:    err.Error(),
			}
		} else {
			progressChan <- TransferProgressMessage{
				FileName:         localPath,
				Status:           "complete",
				BytesTransferred: fileSize,
				TotalBytes:       fileSize,
			}
		}
	}

	return nil
}
