package wallet

import (
	"CocosSDK/crypto/secp256k1"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"math/rand"
	"time"

	"CocosSDK/common/math"
	"CocosSDK/crypto/base58-go"
)

type PrivateKey struct {
	PrivKey   []byte
	VerifySum []byte
}

var VERSION []byte = []byte{0x80}

func GetRandomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.Shuffle(len(bytes), func(i, j int) {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	})
	var r *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func CreatePrivateKey() PrivateKey {
	//str := time.Now().String()
	h := sha256.New()
	h.Write([]byte(GetRandomString(32)))
	sum := h.Sum(nil)
	checkSum := sha256.Sum256(sum)
	return PrivateKey{sum, checkSum[:4]}
}

func CreatePrivateKeyFromSeed(seed string) PrivateKey {
	h := sha256.New()
	h.Write([]byte(seed))
	sum := h.Sum(nil)
	checkSum := sha256.Sum256(sum)
	return PrivateKey{sum, checkSum[:4]}
}

func (prk PrivateKey) ToHexString() string {
	return hex.EncodeToString(prk.PrivKey)
}

func (prk PrivateKey) ToBase58String() string {
	data1 := append(VERSION,
		append(prk.PrivKey, prk.VerifySum...)...)
	bi1 := new(big.Int).SetBytes(data1).String()
	encoded1, _ := base58.BitcoinEncoding.Encode([]byte(bi1))
	return string(encoded1)
}

func PrkFromBase58String(base58Prk string) PrivateKey {
	bytes, _ := base58.BitcoinEncoding.Decode([]byte(base58Prk))
	x, _ := new(big.Int).SetString(string(bytes), 10)
	buf := x.Bytes()
	var prk [32]byte
	copy(prk[:], buf[1:len(buf)-4])
	return PrivateKey{prk[:], buf[len(buf)-4:]}
}

func (prk PrivateKey) GetInt() *big.Int {
	return new(big.Int).SetBytes(prk.PrivKey)
}

func PrkFromWifString(wif string) PrivateKey {
	wif_bytes := Base58Decode([]byte(wif))
	base58_bytes := Base58CheckEncode(VERSION, wif_bytes[:len(wif_bytes)-4])
	bytes, _ := base58.BitcoinEncoding.Decode(base58_bytes)
	x, _ := new(big.Int).SetString(string(bytes), 10)
	buf := x.Bytes()
	var prk [32]byte
	copy(prk[:], buf[1:len(buf)-4])
	return PrivateKey{prk[:], buf[len(buf)-4:]}
}
func (prk PrivateKey) GetUnCompressedPubkey() PublicKey {
	c := secp256k1.S256()
	byte_s := append([]byte{0}, prk.PrivKey...)
	ret := new(big.Int).SetBytes(byte_s)
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = ret
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(ret.Bytes())
	pubkey := append([]byte{4},
		append(priv.PublicKey.X.Bytes(),
			priv.PublicKey.Y.Bytes()...)...)

	return pubkey
}

func (prk PrivateKey) ToEcdsa() *ecdsa.PrivateKey {
	c := secp256k1.S256()
	byte_s := append([]byte{0}, prk.PrivKey...)
	ret := new(big.Int).SetBytes(byte_s)
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = ret
	return priv
}

func (prk PrivateKey) GetSeckey() []byte {
	priv := prk.ToEcdsa()
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

func (prk PrivateKey) GetPublicKey() PublicKey {
	c := secp256k1.S256()
	byte_s := append([]byte{0}, prk.PrivKey...)
	ret := new(big.Int).SetBytes(byte_s)
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = ret
	priv.PublicKey.X, priv.PublicKey.Y = c.ScalarBaseMult(ret.Bytes())
	pubkey := secp256k1.PubkeyFromSeckey(prk.GetSeckey())
	return pubkey
}

func (prk PrivateKey) Sign(data []byte) string {
	for {
		sign := secp256k1.Sign(data, prk.GetSeckey())
		if is_valid(sign) && secp256k1.VerifySignature(data, sign, prk.GetPublicKey()) {
			return hex.EncodeToString(append([]byte{0x1f + sign[64]}, sign[0:64]...))
		}
	}

}

func VerifySignature(data, signature, puk string) bool {
	if data_bytes, err := hex.DecodeString(data); err == nil {
		data_digest := sha256digest(data_bytes)

		if sign, err := hex.DecodeString(signature); err == nil {
			if len(sign) < 65 {
				return false
			}
			if key := PukFromBase58String(puk); key != nil {
				if is_valid(sign[1:]) && secp256k1.VerifySignature(data_digest, append(sign[1:65], sign[0]-0x1f), key) {
					return true
				}
			}
		}
	}
	return false

}

/*验证签名是否是有效的签名*/
func is_valid(sign []byte) bool {
	if sign[0] < 0x80 &&
		(sign[0] != 0x00 || sign[1] > 0x80) &&
		sign[32] < 0x80 &&
		(sign[32] != 0x00 || sign[33] > 0x80) {
		return true
	}
	return false
}

func sha256digest(data []byte) []byte {
	sha := sha256.New()
	sha.Write(data)
	return sha.Sum(nil)
}
