package cy

import (
	"context"

	"github.com/cfoust/cy/pkg/janet"
	"github.com/cfoust/cy/pkg/wm"

	"github.com/rs/zerolog/log"
)

import _ "embed"

//go:embed cy-boot.janet
var CY_BOOT_FILE []byte

func (c *Cy) initJanet(ctx context.Context, configFile string) (*janet.VM, error) {
	vm, err := janet.New(ctx)
	if err != nil {
		return nil, err
	}

	callbacks := map[string]interface{}{
		"log": func(text string) {
			log.Info().Msgf(text)
		},
		"key/bind": func(sequence []string, doc string, callback *janet.Function) error {
			c.tree.Root().Binds().Set(
				sequence,
				wm.Binding{
					Description: doc,
					Callback:    callback,
				},
			)

			return nil
		},
		"pane/current": func(context interface{}) string {
			client, ok := context.(*Client)
			if !ok {
				return ""
			}

			node := client.node
			if node == nil {
				return ""
			}

			_, ok = node.(*wm.Pane)

			return "ok"
		},
	}

	for name, callback := range callbacks {
		err := vm.Callback(name, callback)
		if err != nil {
			return nil, err
		}
	}

	err = vm.ExecuteCall(janet.Call{
		Code:       CY_BOOT_FILE,
		SourcePath: "cy-boot.janet",
		Options:    janet.DEFAULT_CALL_OPTIONS,
	})
	if err != nil {
		return nil, err
	}

	if len(configFile) != 0 {
		err := vm.ExecuteFile(configFile)
		if err != nil {
			return nil, err
		}
	}

	return vm, nil
}
