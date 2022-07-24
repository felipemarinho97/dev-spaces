package ssh

import (
	"fmt"

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
	// connect ot ssh server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		return nil, err
	}

	return &SSHClient{
		host:        host,
		port:        port,
		user:        user,
		identityKey: identityKey,
		conn:        conn,
	}, nil
}

func (c *SSHClient) Run(cmd string) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", err
	}
	out, err := session.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
