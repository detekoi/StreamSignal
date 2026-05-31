//go:build windows

package wincred

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"

	"StreamSignal/internal/ports"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
	errorNotFound           = syscall.Errno(1168)
)

type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWriteW = advapi32.NewProc("CredWriteW")
	procCredReadW  = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

type Store struct{}

func NewStore() ports.SecretStore {
	return &Store{}
}

func (s *Store) Get(_ context.Context, key string) (string, error) {
	targetName, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return "", fmt.Errorf("encode secret key: %w", err)
	}

	var cred *credential
	result, _, callErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&cred)),
	)
	if result == 0 {
		if callErr == errorNotFound {
			return "", fmt.Errorf("secret %q not found", key)
		}
		return "", fmt.Errorf("read Windows credential %q: %w", key, callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))

	if cred.CredentialBlobSize == 0 || cred.CredentialBlob == nil {
		return "", nil
	}

	blob := unsafe.Slice(cred.CredentialBlob, cred.CredentialBlobSize)
	return string(blob), nil
}

func (s *Store) Put(_ context.Context, key string, value string) error {
	targetName, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return fmt.Errorf("encode secret key: %w", err)
	}

	var blobPtr *byte
	blob := []byte(value)
	if len(blob) > 0 {
		blobPtr = &blob[0]
	}

	cred := credential{
		Type:               credTypeGeneric,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     blobPtr,
		Persist:            credPersistLocalMachine,
	}

	result, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if result == 0 {
		return fmt.Errorf("write Windows credential %q: %w", key, callErr)
	}

	return nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	targetName, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return fmt.Errorf("encode secret key: %w", err)
	}

	result, _, callErr := procCredDelete.Call(
		uintptr(unsafe.Pointer(targetName)),
		uintptr(credTypeGeneric),
		0,
	)
	if result == 0 && callErr != errorNotFound {
		return fmt.Errorf("delete Windows credential %q: %w", key, callErr)
	}

	return nil
}
