package ssh

import (
	"fmt"
	"io"
	"net"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
)

type SSHCommand struct {
	Path   string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type SSHClientInterface interface {
	RunCommand(cmd *SSHCommand) ([]byte, error)
	NewSession() (*ssh.Session, *ssh.Client, error)
}

type SSHClient struct {
	Config *ssh.ClientConfig
	Host   string
	Port   int32
}

func (client *SSHClient) RunCommand(cmd *SSHCommand) (out []byte, err error) {
	session, connection, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	defer func() {
		err = session.Close()
		if errors.Is(err, io.EOF) {
			// ignore EOF error
			// on Session Close,
			err = nil
		}

		err = multierr.Append(err, errors.Wrap(err, "failed to close SSH session"))
		err = multierr.Append(err, errors.Wrap(connection.Close(), "failed to close SSH connection"))
	}()

	out, err = session.Output(cmd.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to exec cmd %q on the remote host, %w", cmd.Path, err)
	}

	return
}

func (client *SSHClient) NewSession() (*ssh.Session, *ssh.Client, error) {
	addr := fmt.Sprintf("%s:%d", client.Host, client.Port)

	connection, err := ssh.Dial("tcp", addr, client.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client connection to the SSH server: %s, %w", addr, err)
	}

	session, err := connection.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a new session for a ssh client, %w", err)
	}

	return session, connection, nil
}

func SshInit(userName string, privateKey []byte, host string, port int32, log logr.Logger) (SSHClient, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return SSHClient{}, fmt.Errorf("failed to parse private key for user %q: %w", userName, err)
	}

	sshConfig := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		}),
	}

	newClient := &SSHClient{
		Config: sshConfig,
		Host:   host,
		Port:   port,
	}

	log.Info("SSH Client has been initialized", "Username", userName, "host", host, "port", port)

	return *newClient, nil
}
