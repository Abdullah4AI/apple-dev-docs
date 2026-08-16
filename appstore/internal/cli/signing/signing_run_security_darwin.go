//go:build darwin && cgo

package signing

/*
#cgo CFLAGS: -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>

static OSStatus asc_signing_keychain_create(
    const char *path,
    const unsigned char *password,
    size_t password_length
) {
    SecKeychainRef keychain = NULL;
    OSStatus status = SecKeychainCreate(
        path,
        (UInt32)password_length,
        password,
        false,
        NULL,
        &keychain
    );
    if (status == errSecSuccess) {
        status = SecKeychainUnlock(
            keychain,
            (UInt32)password_length,
            password,
            true
        );
    }
    if (keychain != NULL) {
        CFRelease(keychain);
    }
    return status;
}

static OSStatus asc_signing_keychain_import_pkcs12(
    const char *keychain_path,
    const unsigned char *pkcs12_data,
    size_t pkcs12_length,
    const unsigned char *pkcs12_password,
    size_t pkcs12_password_length,
    const char *trusted_application_path
) {
    OSStatus status = errSecSuccess;
    SecKeychainRef keychain = NULL;
    CFDataRef data = NULL;
    CFStringRef passphrase = NULL;
    SecTrustedApplicationRef application = NULL;
    CFArrayRef applications = NULL;
    SecAccessRef access = NULL;
    CFArrayRef items = NULL;

    status = SecKeychainOpen(keychain_path, &keychain);
    if (status != errSecSuccess) goto cleanup;

    data = CFDataCreate(NULL, pkcs12_data, (CFIndex)pkcs12_length);
    if (data == NULL) { status = errSecAllocate; goto cleanup; }

    passphrase = CFStringCreateWithBytes(
        NULL,
        pkcs12_password,
        (CFIndex)pkcs12_password_length,
        kCFStringEncodingUTF8,
        false
    );
    if (passphrase == NULL) { status = errSecAllocate; goto cleanup; }

    status = SecTrustedApplicationCreateFromPath(trusted_application_path, &application);
    if (status != errSecSuccess) goto cleanup;
    const void *values[] = { application };
    applications = CFArrayCreate(NULL, values, 1, &kCFTypeArrayCallBacks);
    if (applications == NULL) { status = errSecAllocate; goto cleanup; }
    status = SecAccessCreate(CFSTR("ASC ephemeral signing"), applications, &access);
    if (status != errSecSuccess) goto cleanup;

    SecItemImportExportKeyParameters parameters = {
        SEC_KEY_IMPORT_EXPORT_PARAMS_VERSION,
        0,
        passphrase,
        NULL,
        NULL,
        access
    };
    SecExternalFormat format = kSecFormatPKCS12;
    SecExternalItemType type = kSecItemTypeAggregate;
    status = SecItemImport(
        data,
        NULL,
        &format,
        &type,
        0,
        &parameters,
        keychain,
        &items
    );

cleanup:
    if (items != NULL) CFRelease(items);
    if (access != NULL) CFRelease(access);
    if (applications != NULL) CFRelease(applications);
    if (application != NULL) CFRelease(application);
    if (passphrase != NULL) CFRelease(passphrase);
    if (data != NULL) CFRelease(data);
    if (keychain != NULL) CFRelease(keychain);
    return status;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func signingRunSecurityAvailable() bool { return true }

func createKeychainWithSecurityFramework(path string, password []byte) error {
	if len(password) == 0 {
		return fmt.Errorf("keychain password is empty")
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	status := C.asc_signing_keychain_create(
		cPath,
		(*C.uchar)(unsafe.Pointer(&password[0])),
		C.size_t(len(password)),
	)
	if status != 0 {
		return fmt.Errorf("security framework status %d", int32(status))
	}
	return nil
}

func importPKCS12WithSecurityFramework(keychainPath string, data, password []byte) error {
	if len(data) == 0 || len(password) == 0 {
		return fmt.Errorf("PKCS#12 data or password is empty")
	}
	cKeychainPath := C.CString(keychainPath)
	cCodesignPath := C.CString("/usr/bin/codesign")
	defer C.free(unsafe.Pointer(cKeychainPath))
	defer C.free(unsafe.Pointer(cCodesignPath))
	status := C.asc_signing_keychain_import_pkcs12(
		cKeychainPath,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		(*C.uchar)(unsafe.Pointer(&password[0])),
		C.size_t(len(password)),
		cCodesignPath,
	)
	if status != 0 {
		return fmt.Errorf("security framework status %d", int32(status))
	}
	return nil
}
