package ssh

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	host        string
	port        int
	user        string
	identityKey string
	conn        *ssh.Client
}

func NewSSHClient(host string, port int, user string, identityKey string) (*SSHClient, error) {
	signer, err := ssh.ParsePrivateKey([]byte(identityKey))
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	agentDialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 5 * time.Second,
	}
	conn, err := agentDialer.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		conn.Close()
		return nil, err
	}
	err = conn.SetDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		conn.Close()
		return nil, err
	}

	// connect ot ssh server
	clientConn, channelCh, reqCh, err := ssh.NewClientConn(conn, "tcp", config)
	if err != nil {
		return nil, err
	}

	return &SSHClient{
		host:        host,
		port:        port,
		user:        user,
		identityKey: identityKey,
		conn:        ssh.NewClient(clientConn, channelCh, reqCh),
	}, nil
}

func (c *SSHClient) Run(cmd string, timeout time.Duration) (string, error) {
	outCh := make(chan string)
	errCh := make(chan error)
	go func() {
		session, err := c.conn.NewSession()
		if err != nil {
			errCh <- err
			return
		}
		out, err := session.Output(cmd)
		if err != nil {
			errCh <- err
			return
		}

		outCh <- string(out)
	}()

	select {
	case <-time.After(timeout):
		return "", fmt.Errorf("Timeout after %s", timeout)
	case out := <-outCh:
		return out, nil
	case err := <-errCh:
		return "", err
	}
}
