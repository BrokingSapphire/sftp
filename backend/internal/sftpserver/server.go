// Package sftpserver exposes a native SFTP-over-SSH endpoint backed by the same
// storage engine, database and accounts as the REST API. Clients authenticate
// with their username + password or an API key (as the password), and see a
// per-user virtual filesystem isolated to their own files.
package sftpserver

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"sapphirebroking.com/sftp_service/internal/config"
	apikeysvc "sapphirebroking.com/sftp_service/internal/service/apikey"
	authsvc "sapphirebroking.com/sftp_service/internal/service/auth"
	filesvc "sapphirebroking.com/sftp_service/internal/service/file"
	keygen "sapphirebroking.com/sftp_service/pkg/apikey"
	"sapphirebroking.com/sftp_service/pkg/logger"
)

// Deps are the SFTP server dependencies.
type Deps struct {
	Config  config.SFTPConfig
	Auth    *authsvc.Service
	APIKey  *apikeysvc.Service
	Files   *filesvc.Service
	Logger  logger.Logger
}

// Server is the SSH/SFTP listener.
type Server struct {
	cfg      config.SFTPConfig
	auth     *authsvc.Service
	apiKey   *apikeysvc.Service
	files    *filesvc.Service
	log      logger.Logger
	ssh      *ssh.ServerConfig
	listener net.Listener
}

// New builds the SFTP server (does not start listening).
func New(d Deps) (*Server, error) {
	s := &Server{
		cfg: d.Config, auth: d.Auth, apiKey: d.APIKey, files: d.Files,
		log: d.Logger.Named("sftp"),
	}

	hostKey, err := loadOrCreateHostKey(d.Config.HostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("host key: %w", err)
	}

	s.ssh = &ssh.ServerConfig{PasswordCallback: s.passwordCallback}
	s.ssh.AddHostKey(hostKey)
	return s, nil
}

// Start begins accepting SFTP connections; blocks until the listener closes.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	s.listener = ln
	s.log.Info("sftp server listening", "addr", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.log.Warn("sftp accept failed", "err", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// Close stops the listener.
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	ctx := contextBg()
	ip := hostOnly(conn.RemoteAddr().String())
	pw := string(password)

	// API key presented as the password.
	if keygen.Valid(pw) {
		if p, err := s.apiKey.Authenticate(ctx, pw, ip); err == nil {
			return perms(p.UserID), nil
		}
	}
	// Username + password.
	uid, err := s.auth.VerifyPassword(ctx, conn.User(), pw)
	if err != nil {
		s.log.Warn("sftp auth failed", "user", conn.User(), "ip", ip)
		return nil, fmt.Errorf("authentication failed")
	}
	return perms(uid), nil
}

func (s *Server) handleConn(nConn net.Conn) {
	defer nConn.Close()
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, s.ssh)
	if err != nil {
		s.log.Debug("ssh handshake failed", "err", err)
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)

	uid, err := uuid.Parse(sconn.Permissions.Extensions["user_id"])
	if err != nil {
		return
	}

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		channel, requests, err := newChan.Accept()
		if err != nil {
			continue
		}
		go s.serveSession(channel, requests, uid)
	}
}

func (s *Server) serveSession(channel ssh.Channel, requests <-chan *ssh.Request, uid uuid.UUID) {
	// Only accept the "sftp" subsystem.
	go func() {
		for req := range requests {
			ok := req.Type == "subsystem" && len(req.Payload) >= 4 && string(req.Payload[4:]) == "sftp"
			_ = req.Reply(ok, nil)
		}
	}()

	handlers := s.newHandlers(uid)
	server := sftp.NewRequestServer(channel, handlers)
	if err := server.Serve(); err != nil && !errors.Is(err, sftp.ErrSSHFxConnectionLost) {
		s.log.Debug("sftp session ended", "err", err)
	}
	_ = server.Close()
}

func perms(uid uuid.UUID) *ssh.Permissions {
	return &ssh.Permissions{Extensions: map[string]string{"user_id": uid.String()}}
}

func loadOrCreateHostKey(path string) (ssh.Signer, error) {
	if data, err := os.ReadFile(path); err == nil {
		return ssh.ParsePrivateKey(data)
	}
	pem, err := generateHostKey(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(pem)
}

func hostOnly(addr string) string {
	if h, _, err := net.SplitHostPort(addr); err == nil {
		return h
	}
	return addr
}
