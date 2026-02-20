//go:build darwin

package license

/*
#cgo darwin LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static void us_release_cf(CFTypeRef obj) {
	if (obj != NULL) {
		CFRelease(obj);
	}
}

static char* us_cfstring_copy_cstr(CFStringRef s) {
	if (s == NULL) {
		return NULL;
	}
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(CFStringGetLength(s), kCFStringEncodingUTF8) + 1;
	char* out = (char*)malloc((size_t)maxSize);
	if (out == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(s, out, maxSize, kCFStringEncodingUTF8)) {
		free(out);
		return NULL;
	}
	return out;
}

static char* us_cferror_copy_cstr(CFErrorRef err) {
	if (err == NULL) {
		return NULL;
	}
	CFStringRef desc = CFErrorCopyDescription(err);
	char* out = us_cfstring_copy_cstr(desc);
	if (desc != NULL) {
		CFRelease(desc);
	}
	return out;
}

static char* us_osstatus_copy_cstr(OSStatus status) {
	CFStringRef msg = SecCopyErrorMessageString(status, NULL);
	char* out = us_cfstring_copy_cstr(msg);
	if (msg != NULL) {
		CFRelease(msg);
	}
	return out;
}

static CFDataRef us_tag_data(const char* tag) {
	if (tag == NULL) {
		return NULL;
	}
	size_t n = strlen(tag);
	return CFDataCreate(kCFAllocatorDefault, (const UInt8*)tag, (CFIndex)n);
}

static OSStatus us_load_or_create_private_key(const char* tag, int createIfMissing, SecKeyRef* outKey, CFErrorRef* outErr) {
	if (outKey == NULL) {
		return errSecParam;
	}
	*outKey = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	CFDataRef tagData = us_tag_data(tag);
	if (tagData == NULL) {
		return errSecParam;
	}

	CFMutableDictionaryRef query = CFDictionaryCreateMutable(kCFAllocatorDefault, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	if (query == NULL) {
		CFRelease(tagData);
		return errSecAllocate;
	}

	CFDictionarySetValue(query, kSecClass, kSecClassKey);
	CFDictionarySetValue(query, kSecAttrKeyType, kSecAttrKeyTypeECSECPrimeRandom);
	CFDictionarySetValue(query, kSecAttrApplicationTag, tagData);
	CFDictionarySetValue(query, kSecAttrKeyClass, kSecAttrKeyClassPrivate);
	CFDictionarySetValue(query, kSecReturnRef, kCFBooleanTrue);

	OSStatus status = SecItemCopyMatching(query, (CFTypeRef*)outKey);
	if (status == errSecSuccess) {
		CFRelease(query);
		CFRelease(tagData);
		return errSecSuccess;
	}
	if (status != errSecItemNotFound || !createIfMissing) {
		CFRelease(query);
		CFRelease(tagData);
		return status;
	}

	CFMutableDictionaryRef privateAttrs = CFDictionaryCreateMutable(kCFAllocatorDefault, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(kCFAllocatorDefault, 0, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	if (privateAttrs == NULL || attrs == NULL) {
		if (privateAttrs != NULL) CFRelease(privateAttrs);
		if (attrs != NULL) CFRelease(attrs);
		CFRelease(query);
		CFRelease(tagData);
		return errSecAllocate;
	}

	CFErrorRef acErr = NULL;
	SecAccessControlRef ac = SecAccessControlCreateWithFlags(
		kCFAllocatorDefault,
		kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
		kSecAccessControlPrivateKeyUsage,
		&acErr
	);
	if (ac == NULL) {
		if (outErr != NULL) {
			*outErr = acErr;
		} else if (acErr != NULL) {
			CFRelease(acErr);
		}
		CFRelease(privateAttrs);
		CFRelease(attrs);
		CFRelease(query);
		CFRelease(tagData);
		return errSecInternalComponent;
	}

	int keyBits = 256;
	CFNumberRef keyBitsNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &keyBits);
	if (keyBitsNum == NULL) {
		CFRelease(ac);
		CFRelease(privateAttrs);
		CFRelease(attrs);
		CFRelease(query);
		CFRelease(tagData);
		return errSecAllocate;
	}

	CFDictionarySetValue(privateAttrs, kSecAttrIsPermanent, kCFBooleanTrue);
	CFDictionarySetValue(privateAttrs, kSecAttrApplicationTag, tagData);
	CFDictionarySetValue(privateAttrs, kSecAttrAccessControl, ac);

	CFDictionarySetValue(attrs, kSecAttrKeyType, kSecAttrKeyTypeECSECPrimeRandom);
	CFDictionarySetValue(attrs, kSecAttrTokenID, kSecAttrTokenIDSecureEnclave);
	CFDictionarySetValue(attrs, kSecAttrKeySizeInBits, keyBitsNum);
	CFDictionarySetValue(attrs, kSecPrivateKeyAttrs, privateAttrs);

	CFErrorRef keyErr = NULL;
	SecKeyRef key = SecKeyCreateRandomKey(attrs, &keyErr);
	if (key == NULL) {
		if (outErr != NULL) {
			*outErr = keyErr;
		} else if (keyErr != NULL) {
			CFRelease(keyErr);
		}
		CFRelease(keyBitsNum);
		CFRelease(ac);
		CFRelease(privateAttrs);
		CFRelease(attrs);
		CFRelease(query);
		CFRelease(tagData);
		return errSecInternalComponent;
	}

	*outKey = key;
	CFRelease(keyBitsNum);
	CFRelease(ac);
	CFRelease(privateAttrs);
	CFRelease(attrs);
	CFRelease(query);
	CFRelease(tagData);
	return errSecSuccess;
}

static OSStatus us_secure_enclave_public_key_x963(const char* tag, int createIfMissing, CFDataRef* outData, CFErrorRef* outErr) {
	if (outData == NULL) {
		return errSecParam;
	}
	*outData = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	SecKeyRef privateKey = NULL;
	OSStatus status = us_load_or_create_private_key(tag, createIfMissing, &privateKey, outErr);
	if (status != errSecSuccess) {
		return status;
	}

	SecKeyRef publicKey = SecKeyCopyPublicKey(privateKey);
	if (publicKey == NULL) {
		CFRelease(privateKey);
		return errSecInternalComponent;
	}

	CFErrorRef exportErr = NULL;
	CFDataRef raw = SecKeyCopyExternalRepresentation(publicKey, &exportErr);
	if (raw == NULL) {
		if (outErr != NULL) {
			*outErr = exportErr;
		} else if (exportErr != NULL) {
			CFRelease(exportErr);
		}
		CFRelease(publicKey);
		CFRelease(privateKey);
		return errSecInternalComponent;
	}

	*outData = raw;
	CFRelease(publicKey);
	CFRelease(privateKey);
	return errSecSuccess;
}

static OSStatus us_secure_enclave_sign_digest(const char* tag, int createIfMissing, const unsigned char* digest, size_t digestLen, CFDataRef* outSig, CFErrorRef* outErr) {
	if (outSig == NULL || digest == NULL) {
		return errSecParam;
	}
	*outSig = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	SecKeyRef privateKey = NULL;
	OSStatus status = us_load_or_create_private_key(tag, createIfMissing, &privateKey, outErr);
	if (status != errSecSuccess) {
		return status;
	}

	CFDataRef digestData = CFDataCreate(kCFAllocatorDefault, digest, (CFIndex)digestLen);
	if (digestData == NULL) {
		CFRelease(privateKey);
		return errSecAllocate;
	}

	CFErrorRef signErr = NULL;
	CFDataRef sig = SecKeyCreateSignature(
		privateKey,
		kSecKeyAlgorithmECDSASignatureDigestX962SHA256,
		digestData,
		&signErr
	);
	CFRelease(digestData);
	CFRelease(privateKey);

	if (sig == NULL) {
		if (outErr != NULL) {
			*outErr = signErr;
		} else if (signErr != NULL) {
			CFRelease(signErr);
		}
		return errSecInternalComponent;
	}

	*outSig = sig;
	return errSecSuccess;
}
*/
import "C"

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"unsafe"
)

const defaultSecureEnclaveKeyTag = "github.com.bronystylecrazy.ultrastructure.license.secureenclave"

// MacOSKeychainSecureEnclaveSigner provides a concrete SecureEnclaveSigner backed by
// an EC P-256 private key stored in macOS Keychain/Secure Enclave.
type MacOSKeychainSecureEnclaveSigner struct {
	keyTag          string
	createIfMissing bool
}

func NewMacOSKeychainSecureEnclaveSigner(keyTag string, createIfMissing bool) *MacOSKeychainSecureEnclaveSigner {
	if keyTag == "" {
		keyTag = defaultSecureEnclaveKeyTag
	}
	return &MacOSKeychainSecureEnclaveSigner{
		keyTag:          keyTag,
		createIfMissing: createIfMissing,
	}
}

func (s *MacOSKeychainSecureEnclaveSigner) PublicKeyDER(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tag := C.CString(s.keyTag)
	defer C.free(unsafe.Pointer(tag))

	var outData C.CFDataRef
	var outErr C.CFErrorRef
	status := C.us_secure_enclave_public_key_x963(tag, boolToCInt(s.createIfMissing), &outData, &outErr)
	if status != C.errSecSuccess {
		return nil, seErrorf("load secure enclave public key", status, outErr)
	}
	defer C.us_release_cf(C.CFTypeRef(outData))

	raw := copyCFDataBytes(outData)
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: empty secure enclave public key", ErrDeviceBindingUnavailable)
	}

	pubDER, err := x963ToPKIX(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: convert public key: %v", ErrDeviceBindingUnavailable, err)
	}
	return pubDER, nil
}

func (s *MacOSKeychainSecureEnclaveSigner) SignDigest(ctx context.Context, digest []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(digest) != sha256.Size {
		return nil, fmt.Errorf("%w: digest length must be %d bytes", ErrChallengeFailed, sha256.Size)
	}

	tag := C.CString(s.keyTag)
	defer C.free(unsafe.Pointer(tag))

	var outSig C.CFDataRef
	var outErr C.CFErrorRef
	status := C.us_secure_enclave_sign_digest(
		tag,
		boolToCInt(s.createIfMissing),
		(*C.uchar)(unsafe.Pointer(&digest[0])),
		C.size_t(len(digest)),
		&outSig,
		&outErr,
	)
	if status != C.errSecSuccess {
		return nil, seErrorf("sign challenge digest", status, outErr)
	}
	defer C.us_release_cf(C.CFTypeRef(outSig))

	signature := copyCFDataBytes(outSig)
	if len(signature) == 0 {
		return nil, fmt.Errorf("%w: empty secure enclave signature", ErrChallengeFailed)
	}
	return signature, nil
}

func x963ToPKIX(raw []byte) ([]byte, error) {
	x, y := elliptic.Unmarshal(elliptic.P256(), raw)
	if x == nil || y == nil {
		return nil, errors.New("invalid x963 EC point")
	}
	pub := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
	return x509.MarshalPKIXPublicKey(pub)
}

func seErrorf(action string, status C.OSStatus, cfErr C.CFErrorRef) error {
	if cfErr != 0 {
		msg := cStringAndFree(C.us_cferror_copy_cstr(cfErr))
		C.us_release_cf(C.CFTypeRef(cfErr))
		if msg != "" {
			return fmt.Errorf("%w: %s: %s", ErrDeviceBindingUnavailable, action, msg)
		}
	}

	statusMsg := cStringAndFree(C.us_osstatus_copy_cstr(status))
	if statusMsg != "" {
		return fmt.Errorf("%w: %s: osstatus=%d (%s)", ErrDeviceBindingUnavailable, action, int(status), statusMsg)
	}
	return fmt.Errorf("%w: %s: osstatus=%d", ErrDeviceBindingUnavailable, action, int(status))
}

func copyCFDataBytes(data C.CFDataRef) []byte {
	n := int(C.CFDataGetLength(data))
	if n <= 0 {
		return nil
	}
	ptr := C.CFDataGetBytePtr(data)
	if ptr == nil {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(ptr), C.int(n))
}

func cStringAndFree(cs *C.char) string {
	if cs == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

func boolToCInt(v bool) C.int {
	if v {
		return 1
	}
	return 0
}
