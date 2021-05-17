package goinept

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"

	"github.com/miramaris/goinept/internal/zip"
	"golang.org/x/sync/errgroup"
)

type Rights struct {
	XMLName      xml.Name `xml:"rights"`
	LicenseToken struct {
		EncryptedKey string `xml:"encryptedKey"`
	} `xml:"licenseToken"`
}

type Encryption struct {
	XMLName       xml.Name `xml:"encryption"`
	Text          string   `xml:",chardata"`
	Xmlns         string   `xml:"xmlns,attr"`
	EncryptedData []struct {
		Text             string `xml:",chardata"`
		Xmlns            string `xml:"xmlns,attr"`
		EncryptionMethod struct {
			Text      string `xml:",chardata"`
			Algorithm string `xml:"Algorithm,attr"`
		} `xml:"EncryptionMethod"`
		KeyInfo struct {
			Text     string `xml:",chardata"`
			Xmlns    string `xml:"xmlns,attr"`
			Resource struct {
				Text  string `xml:",chardata"`
				Xmlns string `xml:"xmlns,attr"`
			} `xml:"resource"`
		} `xml:"KeyInfo"`
		CipherData struct {
			Text            string `xml:",chardata"`
			CipherReference struct {
				Text string `xml:",chardata"`
				URI  string `xml:"URI,attr"`
			} `xml:"CipherReference"`
		} `xml:"CipherData"`
	} `xml:"EncryptedData"`
}

type Decryptor struct {
	Key       []byte
	Encrypted []string
	Block     cipher.Block
}

type JobResult struct {
	Decrypted []byte
	Header    *zip.FileHeader
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (d *Decryptor) Init(e []byte) error {
	var encryption Encryption
	err := xml.Unmarshal(e, &encryption)
	if err != nil {
		return err
	}

	for _, f := range encryption.EncryptedData {
		if f.CipherData.CipherReference.URI != "" {
			d.Encrypted = append(d.Encrypted, f.CipherData.CipherReference.URI)
		}
	}

	block, err := aes.NewCipher(d.Key)
	if err != nil {
		return err
	}
	d.Block = block

	return nil
}

func (d *Decryptor) Decrypt(data []byte) ([]byte, error) {
	iv := data[:16]
	ciphertext := data[16:]
	mode := cipher.NewCBCDecrypter(d.Block, iv)
	tmp := make([]byte, len(ciphertext))
	copy(tmp, ciphertext)
	mode.CryptBlocks(tmp, tmp)

	decompressed, err := Decompress(tmp[:len(tmp)-int(tmp[len(tmp)-1])])
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}

func Decompress(data []byte) ([]byte, error) {
	buff := bytes.NewReader(data)
	r := flate.NewReader(buff)
	defer r.Close()

	d, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func ReadZipFile(f *zip.File) ([]byte, error) {
	h, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer h.Close()

	c, err := ioutil.ReadAll(h)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func DecodeBookkey(privatekey *rsa.PrivateKey, k string) ([]byte, error) {
	bookkey, err := base64.StdEncoding.DecodeString(k)
	if err != nil {
		return nil, err
	}

	c := new(big.Int).SetBytes(bookkey)
	b := c.Exp(c, privatekey.D, privatekey.N).Bytes()
	return b, nil
}

func determineFileName(f *zip.File) string {
	lh, err := f.ReadLocalHeader()
	if err != nil {
		log.Fatal(err)
	}
	localName := lh.Name
	centralName := f.Name
	
	if localName == "" && centralName != "" {
		return centralName
	}

	return localName
}

func DecryptEpub(keyFilepath string, epubFilepath string, outputFilepath string) {
	keyFileBytes, err := ioutil.ReadFile(keyFilepath)
	if err != nil {
		log.Fatal(err)
	}

	epubFileBytes, err := ioutil.ReadFile(epubFilepath)
	if err != nil {
		log.Fatal(err)
	}

	buf := DecryptEpubFromBytes(keyFileBytes, epubFileBytes)

	o, err := os.OpenFile(outputFilepath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal("error creating output file: ", err)
	}
	defer o.Close()

	io.Copy(o, buf)
}

func DecryptEpubFromBytes(keyFile []byte, epubFile []byte) *bytes.Buffer {
	privateKey, err := x509.ParsePKCS1PrivateKey(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	zipReader := bytes.NewReader(epubFile)
	inf, err := zip.NewReader(zipReader, int64(len(epubFile)))
	if err != nil {
		log.Fatal(err)
	}

	hasRights := false
	hasEncryption := false

	zipFiles := make(map[string]*zip.File, len(inf.File))
	for _, f := range inf.File {
		n := determineFileName(f)
		zipFiles[n] = f
		if n == "META-INF/rights.xml" {
			hasRights = true
		}
		if n == "META-INF/encryption.xml" {
			hasEncryption = true
		}
	}

	if !hasRights || !hasEncryption {
		log.Fatal("file is DRM-free.")
	}

	namelist := make([]string, 0, len(zipFiles))
	for k := range zipFiles {
		if k != "META-INF/rights.xml" && k != "META-INF/encryption.xml" && k != "mimetype" {
			namelist = append(namelist, k)
		}
	}

	rightsContent, err := ReadZipFile(zipFiles["META-INF/rights.xml"])
	if err != nil {
		log.Fatal(err)
	}

	var rights Rights
	err = xml.Unmarshal(rightsContent, &rights)
	if err != nil {
		log.Fatal(err)
	}

	if len(rights.LicenseToken.EncryptedKey) != 172 {
		log.Fatal("file is not a secure Adobe Adept ePub.")
	}

	bookkey, err := DecodeBookkey(privateKey, rights.LicenseToken.EncryptedKey)
	if err != nil {
		log.Fatal(err)
	}

	if bookkey[len(bookkey)-17] != byte(0) {
		log.Fatal("Could not decrypt. Wrong key.")
	}

	encryptionContent, err := ReadZipFile(zipFiles["META-INF/encryption.xml"])
	if err != nil {
		log.Fatal(err)
	}

	decryptor := Decryptor{Key: bookkey[len(bookkey)-16:]}
	err = decryptor.Init(encryptionContent)
	if err != nil {
		log.Fatal(err)
	}

	buf := new(bytes.Buffer)
	outf := zip.NewWriter(buf)

	mimetype := zipFiles["mimetype"]
	newHeader := zip.FileHeader{
		Name:           mimetype.FileHeader.Name,
		Comment:        mimetype.FileHeader.Comment,
		CreatorVersion: mimetype.FileHeader.CreatorVersion,
		Flags:          mimetype.FileHeader.Flags,
		Method:         mimetype.FileHeader.Method,
		ModifiedDate:   mimetype.FileHeader.ModifiedDate,
		ModifiedTime:   mimetype.FileHeader.ModifiedTime,
		ExternalAttrs:  mimetype.FileHeader.ExternalAttrs,
	}
	w, err := outf.CreateHeader(&newHeader)
	if err != nil {
		log.Fatal("error creating zip header", err)
	}

	f, err := mimetype.Open()
	if err != nil {
		log.Fatal("error opening mimetype file", err)
	}

	_, err = io.Copy(w, f)
	if err != nil {
		log.Fatal("error writing mimetype zip header", err)
	}

	err = f.Close()
	if err != nil {
		log.Fatal("error mimetype file", err)
	}

	jobs := make(chan *zip.File, len(namelist))
	results := make(chan JobResult, len(namelist))

	g, ctx := errgroup.WithContext(context.Background())

	block, err := aes.NewCipher(bookkey[len(bookkey)-16:])
	if err != nil {
		log.Fatal(err)
	}

	g.Go(func() error {
		for f := range jobs {
			data, err := ReadZipFile(f)
			if err != nil {
				return err
			}

			iv := data[:16]
			ciphertext := data[16:]
			mode := cipher.NewCBCDecrypter(block, iv)
			tmp := make([]byte, len(ciphertext))
			mode.CryptBlocks(tmp, ciphertext)

			dec, err := Decompress(tmp[:len(tmp)-int(tmp[len(tmp)-1])])
			if err != nil {
				return err
			}

			n := determineFileName(f)
			h := &zip.FileHeader{
				Name:           n,
				Comment:        f.FileHeader.Comment,
				CreatorVersion: f.FileHeader.CreatorVersion,
				Flags:          f.FileHeader.Flags,
				Method:         zip.Deflate,
				ModifiedDate:   f.FileHeader.ModifiedDate,
				ModifiedTime:   f.FileHeader.ModifiedTime,
				Extra:          f.FileHeader.Extra,
				ExternalAttrs:  f.FileHeader.ExternalAttrs,
			}

			result := JobResult{dec, h}

			select {
			case results <- result:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})

	for _, path := range namelist {
		f := zipFiles[path]
		_, encrypted := Find(decryptor.Encrypted, path)
		if encrypted {
			jobs <- f
		} else {
			data, err := ReadZipFile(f)
			if err != nil {
				close(jobs)
				log.Fatal("error reading zip file: ", err)
			}

			n := determineFileName(f)
			h := &zip.FileHeader{
				Name:           n,
				Comment:        f.FileHeader.Comment,
				CreatorVersion: f.FileHeader.CreatorVersion,
				Flags:          f.FileHeader.Flags,
				Method:         zip.Deflate,
				ModifiedDate:   f.FileHeader.ModifiedDate,
				ModifiedTime:   f.FileHeader.ModifiedTime,
				Extra:          f.FileHeader.Extra,
				ExternalAttrs:  f.FileHeader.ExternalAttrs,
			}

			result := JobResult{data, h}
			results <- result
		}
	}
	close(jobs)

	go func() {
		g.Wait()
		close(results)
	}()

	for result := range results {
		w, err := outf.CreateHeader(result.Header)
		if err != nil {
			log.Fatal("error writing zip file for file: ", err)
		}

		_, err = io.Copy(w, bytes.NewReader(result.Decrypted))
		if err != nil {
			log.Printf("%+v", result.Header)
			log.Fatal("error writing zip header: ", err)
		}
	}

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

	err = outf.Close()
	if err != nil {
		log.Fatal("error closing zip file: ", err)
	}

	return buf
}
