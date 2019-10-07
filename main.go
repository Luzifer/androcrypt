package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/go-openssl/v3"
	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		Decrypt        bool   `flag:"decrypt,d" default:"false" description:"Execute decrypt operation"`
		InFile         string `flag:"in-file,i" default:"-" description:"File to read input from (- for stdin)"`
		KDF            string `flag:"kdf" default:"sha256" description:"Key-Derivation-Function to use (on of md5, sha1, sha256)"`
		Key            string `flag:"key,k" default:"" description:"Key to use for en- or decryption"`
		KeyFile        string `flag:"key-file,f" default:"" description:"Key-File which content is used instead of the key (overwrites key flag)"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		OutFile        string `flag:"out-file,o" default:"-" description:"File to write output to (- for stdout)"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("git-changerelease %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	if cfg.KeyFile != "" {
		rawKey, err := ioutil.ReadFile(cfg.KeyFile)
		if err != nil {
			log.WithError(err).Fatal("Unable to read key from file")
		}
		cfg.Key = string(rawKey)
	}

	if cfg.Key == "" {
		log.Fatal("No key given")
	}

	var kdf openssl.DigestFunc
	switch cfg.KDF {
	case "md5":
		kdf = openssl.DigestMD5Sum
	case "sha1":
		kdf = openssl.DigestSHA1Sum
	case "sha256":
		kdf = openssl.DigestSHA256Sum
	default:
		log.Fatalf("No such Key-Derivation-Function: %q", cfg.KDF)
	}

	var (
		input  io.Reader
		output io.Writer
	)

	if cfg.InFile == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(cfg.InFile)
		if err != nil {
			log.WithError(err).Fatal("Unable to open input file")
		}
		defer f.Close()
		input = f
	}

	if cfg.OutFile == "-" {
		output = os.Stdout
	} else {
		f, err := os.Create(cfg.OutFile)
		if err != nil {
			log.WithError(err).Fatal("Unable to create output file")
		}
		defer f.Close()
		output = f
	}

	var (
		err error
		in  = new(bytes.Buffer)
		out []byte
	)
	if _, err := io.Copy(in, input); err != nil {
		log.WithError(err).Fatal("Unable to read input file")
	}

	if cfg.Decrypt {
		out, err = openssl.New().DecryptBytes(cfg.Key, in.Bytes(), kdf)
	} else {
		out, err = openssl.New().EncryptBytes(cfg.Key, in.Bytes(), kdf)
	}

	if err != nil {
		log.WithError(err).Fatal("Unable to perform cryptographic action")
	}

	if _, err = output.Write(out); err != nil {
		log.WithError(err).Fatal("Unable to write output data")
	}
}
