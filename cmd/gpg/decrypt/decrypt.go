package decrypt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/clstb/ksp/pkg/injector"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Run executes the decrypt command
func Run(c *cli.Context) error {
	ctx := context.Background()

	file := c.String("file")

	data, err := readFile(file)
	if err != nil {
		return err
	}

	gpg, err := injector.NewGPG(ctx)
	if err != nil {
		return errors.Wrap(err, "creating gpg injector failed")
	}

	for k, v := range data {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return errors.Wrapf(err, "decoding key %s failed", k)
		}

		decrypted, err := gpg.Decrypt(decoded)
		if err != nil {
			return errors.Wrapf(err, "decrypting key %s failed", k)
		}

		data[k] = string(decrypted)
	}

	var b bytes.Buffer
	for k, v := range data {
		_, err := b.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		if err != nil {
			return errors.Wrap(err, "writing decrypted data to buffer failed")
		}
	}

	if err := ioutil.WriteFile(file, b.Bytes(), os.ModePerm); err != nil {
		return errors.Wrap(err, "writing file failed")
	}

	return nil
}

func readFile(path string) (map[string]string, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "opening file failed")
	}
	defer f.Close()

	data := make(map[string]string)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		k := s[:strings.IndexByte(s, '=')]
		v := s[strings.IndexByte(s, '=')+1:]
		data[k] = v
	}

	if scanner.Err() != nil {
		return nil, errors.Wrap(err, "reading file failed")
	}

	return data, nil
}
