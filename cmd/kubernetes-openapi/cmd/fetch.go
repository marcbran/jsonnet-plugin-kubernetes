package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/internal/kubernetesopenapi"
	kubernetesopenapipkg "github.com/marcbran/jsonnet-plugin-kubernetes/cmd/kubernetes-openapi/pkg/kubernetesopenapi"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch CONTEXT...",
	Short: "Fetch the OpenAPI v3 index and per-group/version documents for one or more kubeconfig contexts",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runFetch,
}

func init() {
	fetchCmd.Flags().StringP("out", "o", ".", "output directory")
	fetchCmd.Flags().StringP("format", "f", "text", "Output format: text, json")
}

func runFetch(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	outDir, err := cmd.Flags().GetString("out")
	if err != nil {
		return err
	}
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return err
	}

	if !quiet && format == "text" {
		_, err = fmt.Fprintf(os.Stderr, "fetching openapi/v3 from contexts %q\n", args)
		if err != nil {
			return err
		}
	}

	facade := kubernetesopenapi.NewFacade()
	out, err := facade.FetchOpenAPI(cmd.Context(), kubernetesopenapipkg.FetchOpenAPIInput{
		Contexts: args,
		OutDir:   outDir,
	})
	if err != nil {
		return err
	}

	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(out)
	default:
		for _, gv := range out.GroupVersions {
			_, err = fmt.Fprintln(os.Stdout, gv)
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintln(os.Stderr, out.BundleFile)
		return err
	}
}
