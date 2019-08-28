package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/icon-project/goloop/common/wallet"
	"github.com/spf13/cobra"
)

func main() {
	var keyStoreFile string
	var keyStorePass string
	var scorePath string
	var tps int
	var concurrent int
	var walletCount int
	var nid int64
	var methodName string
	var params map[string]string
	var installParams map[string]string

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s [urls]", os.Args[0]),
	}
	flags := cmd.PersistentFlags()
	flags.StringVarP(&keyStoreFile, "keystore", "k", "", "File path to keystore of base account(like GOD)")
	flags.StringVarP(&keyStorePass, "password", "p", "gochain", "Password for the keystore")
	flags.IntVarP(&tps, "tps", "t", 1000, "Max transaction per a second")
	flags.IntVarP(&concurrent, "concurrent", "c", 2, "Number of subroutines(threads)")
	flags.IntVarP(&walletCount, "wallets", "w", 1000, "Number of temporal wallets")
	flags.Int64VarP(&nid, "nid", "n", 1, "Network ID of URLs")
	flags.StringVarP(&scorePath, "score", "s", "", "Path to SCORE source directory")
	flags.StringVarP(&methodName, "method", "m", "transfer", "Method name to be used for transfer")
	flags.StringToStringVar(&params, "param", nil, "Parameters for the call")
	flags.StringToStringVar(&installParams, "installParam", make(map[string]string), "Install parameters")

	cmd.Run = func(cmd *cobra.Command, urls []string) {
		if len(urls) == 0 {
			urls = []string{"http://localhost:9080/api/v3"}
		}

		if keyStoreFile == "" {
			log.Panic("KeyStore for base account isn't specified")
		}

		ks, err := ioutil.ReadFile(keyStoreFile)
		if err != nil {
			log.Panicf("Fail to read KeyStore file=%s err=%+v", keyStoreFile, err)
		}

		godWallet, err := wallet.NewFromKeyStore(ks, []byte(keyStorePass))
		if err != nil {
			log.Panicf("Fail to decrypt KeyStore err=%+v", err)
		}

		var maker TransactionMaker
		if len(scorePath) > 0 && params != nil {
			maker = &CallMaker{
				NID:           nid,
				SourcePath:    scorePath,
				InstallParams: installParams,
				Method:        methodName,
				CallParams:    params,
				god:           godWallet,
				index:         0,
			}
		} else if len(scorePath) > 0 {
			maker = &TokenTransferMaker{
				NID:         nid,
				WalletCount: walletCount,
				SourcePath:  scorePath,
				Method:      methodName,
			}
		} else {
			maker = &CoinTransferMaker{
				NID:         nid,
				WalletCount: walletCount,
				GodWallet:   godWallet,
			}
		}

		ctx := NewContext(concurrent, int64(tps), maker)
		ctx.Run(urls)
	}

	cmd.Execute()
}
