// Copyright 2019 Adrien Waksberg
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import(
  "crypto/aes"
  "crypto/sha1"
  "crypto/cipher"
  "crypto/rand"
  "encoding/base64"
  "io"

  "golang.org/x/crypto/pbkdf2"
)

// Encrypt data with aes256
func Encrypt(data []byte, passphrase string, salt string) (string, error) {
  key := pbkdf2.Key([]byte(passphrase), []byte(salt), 4096, 32, sha1.New)

  block, err := aes.NewCipher([]byte(key))
  if err != nil {
    return "", err
  }

  cipher, err := cipher.NewGCM(block)
  if err != nil {
    return "", err
  }

  nonce := make([]byte, cipher.NonceSize())
  _, err = io.ReadFull(rand.Reader, nonce)
  if err != nil {
    return "", err
  }

  dataEncrypted := cipher.Seal(nonce, nonce, data, nil)

  return base64.StdEncoding.EncodeToString(dataEncrypted), nil
}

// Decrypt data
func Decrypt(data string, passphrase string, salt string) ([]byte, error) {
  key := pbkdf2.Key([]byte(passphrase), []byte(salt), 4096, 32, sha1.New)

  rawData, err := base64.StdEncoding.DecodeString(data)
  if err != nil {
    return []byte{}, err
  }

  block, err := aes.NewCipher([]byte(key))
  if err != nil {
    return []byte{}, err
  }

  cipher, err := cipher.NewGCM(block)
  if err != nil {
    return []byte{}, err
  }

  nonceSize := cipher.NonceSize()
  nonce, text := rawData[:nonceSize], rawData[nonceSize:]
  plaintext, err := cipher.Open(nil, nonce, text, nil)
  if err != nil {
    return []byte{}, err
  }

  return plaintext, nil
}
