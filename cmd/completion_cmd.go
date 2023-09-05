package cmd

import (
	"os"

	"github.com/daedaleanai/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion bash|zsh|fish",
	Short: "Generate completion script",
	Long: `To load completions:
Bash:
  $ source <(reqtraq completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ reqtraq completion bash > /etc/bash_completion.d/reqtraq
  # macOS:
  $ reqtraq completion bash > /usr/local/etc/bash_completion.d/reqtraq
Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ reqtraq completion zsh > "${fpath[1]}/_reqtraq"
  # You will need to start a new shell for this setup to take effect.
fish:
  $ reqtraq completion fish | source
  # To load completions for each session, execute once:
  $ reqtraq completion fish > ~/.config/fish/completions/reqtraq.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		}
	},
	Hidden: true,
}

// Registers the completion subcommand
// @llr REQ-TRAQ-SWL-57
func init() {
	rootCmd.AddCommand(completionCmd)
}
