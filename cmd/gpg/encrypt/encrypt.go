package encrypt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clstb/ksp/pkg/injector"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// Run executes the encrypt command
func Run(c *cli.Context) error {
	ctx := context.Background()

	keys := c.StringSlice("keys")
	dataString := c.String("data")

	data := make(map[string]string)
	if err := json.NewDecoder(strings.NewReader(dataString)).Decode(&data); err != nil {
		return errors.Wrap(err, "decoding data failed")
	}

	gpg, err := injector.NewGPG(ctx)
	if err != nil {
		return err
	}

	encryptedData := make(map[string]string)
	for k, v := range data {
		b, err := gpg.Encrypt(keys, []byte(v))
		if err != nil {
			return err
		}

		encrypted := base64.StdEncoding.EncodeToString(b)
		encryptedData[k] = encrypted
	}

	var js bytes.Buffer
	if err := json.NewEncoder(&js).Encode(&encryptedData); err != nil {
		return err
	}

	fmt.Println(js.String())

	return nil
}
