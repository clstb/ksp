package encrypt

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

// Run executes the encrypt command
func Run(c *cli.Context) error {
	ctx := context.Background()

	keys := c.StringSlice("keys")
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
		encrypted, err := gpg.Encrypt(keys, []byte(v))
		if err != nil {
			return errors.Wrapf(err, "encrypting key %s failed", k)
		}

		encoded := base64.StdEncoding.EncodeToString(encrypted)
		data[k] = encoded
	}

	var b bytes.Buffer
	for k, v := range data {
		_, err := b.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		if err != nil {
			return errors.Wrap(err, "writing encrypted data failed")
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
	line := 0
	for scanner.Scan() {
		line++

		splitted := strings.Split(scanner.Text(), "=")
		if scanner.Err() != nil {
			return nil, errors.Wrap(err, "reading file failed")
		}
		if len(splitted) > 2 {
			return nil, errors.Errorf("invalid input at line: %d", line)
		}

		k, v := splitted[0], splitted[1]
		data[k] = v
	}

	return data, nil
}
