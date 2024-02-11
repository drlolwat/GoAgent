package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

type ecb struct {
	block     cipher.Block
	blockSize int
}

func newECB(block cipher.Block) *ecb {
	return &ecb{
		block:     block,
		blockSize: block.BlockSize(),
	}
}

type ecbDecrypter ecb

func NewECBDecrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbDecrypter)(newECB(b))
}

func (decrypter *ecbDecrypter) BlockSize() int {
	return decrypter.blockSize
}

func (decrypter *ecbDecrypter) CryptBlocks(dst, src []byte) {
	if len(src)%decrypter.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		decrypter.block.Decrypt(dst, src[:decrypter.blockSize])
		src = src[decrypter.blockSize:]
		dst = dst[decrypter.blockSize:]
	}
}

func pKCS5Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func generateKey(keyHexString string) ([]byte, error) {
	keyBytes, err := hex.DecodeString(keyHexString)
	if err != nil {
		return nil, err
	}

	if len(keyBytes) > 16 {
		keyBytes = keyBytes[:16]
	} else if len(keyBytes) < 16 {
		padded := make([]byte, 16)
		copy(padded, keyBytes)
		keyBytes = padded
	}

	if _, err := aes.NewCipher(keyBytes); err != nil {
		return nil, err
	}

	return keyBytes, nil
}
