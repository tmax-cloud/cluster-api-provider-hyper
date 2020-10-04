package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type remoteScriptType byte
type PassThru func(r io.Reader, total int64) io.Reader

const (
	cmdLine remoteScriptType = iota
	rawScript
	scriptFile
)

type Client struct {
	client *ssh.Client
}

func SSHConnect(addr, user, password string) (*Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		Timeout:         time.Duration(2000 * time.Second),
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return &Client{conn}, nil
}

func (conn *Client) SendCommands(cmds ...string) ([]byte, error) {
	session, err := conn.client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return []byte{}, err
	}

	cmd := strings.Join(cmds, "; ")
	output, err := session.Output(cmd)
	if err != nil {
		return output, err
	}

	return output, err
}

func (c *Client) Close() {
	c.client.Close()
}

// ScriptFile
func (c *Client) ScriptFile(fname string) *remoteScript {
	return &remoteScript{
		_type:      scriptFile,
		client:     c.client,
		scriptFile: fname,
	}
}

type remoteScript struct {
	client     *ssh.Client
	_type      remoteScriptType
	script     *bytes.Buffer
	scriptFile string
	err        error

	stdout io.Writer
	stderr io.Writer
}

// Run
func (rs *remoteScript) Run() error {
	if rs.err != nil {
		fmt.Println(rs.err)
		return rs.err
	}

	if rs._type == scriptFile {
		return rs.runScriptFile()
	} else {
		return errors.New("Not supported remoteScript type")
	}
}

func (rs *remoteScript) Output() ([]byte, error) {
	if rs.stdout != nil {
		return nil, errors.New("Stdout already set")
	}
	var out bytes.Buffer
	rs.stdout = &out
	err := rs.Run()
	return out.Bytes(), err
}

func (rs *remoteScript) runScript() error {
	session, err := rs.client.NewSession()
	if err != nil {
		return err
	}

	session.Stdin = rs.script
	session.Stdout = rs.stdout
	session.Stderr = rs.stderr

	if err := session.Shell(); err != nil {
		return err
	}
	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}

func (rs *remoteScript) runScriptFile() error {
	var buffer bytes.Buffer
	file, err := os.Open(rs.scriptFile)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(&buffer, file)
	if err != nil {
		return err
	}

	rs.script = &buffer
	return rs.runScript()
}

func (c *Client) ScpFile(filePath, remotePath string, configMap map[string]string) error {
	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buff := make([]byte, 1024)
	contents := ""
	for {
		cnt, err := f.Read(buff)
		if err != nil && err != io.EOF {
			panic(err)
		}

		if cnt == 0 {
			break
		}

		contents = contents + string(buff[:cnt])
	}

	for key, val := range configMap {
		contents = strings.Replace(contents, key, val, -1)
	}

	if _, err := c.SendCommands("echo -e \"" + contents + "\" >> " + remotePath); err != nil {
		return err
	}
	return nil
}
