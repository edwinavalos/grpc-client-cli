package main

import (
	"fmt"
	"github.com/spf13/viper"

	prompt "github.com/c-bata/go-prompt"
)

func main() {
	rootLogger := NewLogger()
	viperCfg := viper.New()

	viperCfg.SetConfigName("config")
	viperCfg.AddConfigPath("/etc/tracker")
	viperCfg.AddConfigPath("$HOME/.tracker")
	viperCfg.AddConfigPath(".")
	viperCfg.SetConfigType("yaml")

	err := viperCfg.ReadInConfig()
	if err != nil {
		panic(err)
	}

	cfg := NewConfig()

	err = viperCfg.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	for _, ep := range cfg.Endpoints {
		rootLogger.Infof("Endpoints from config: %s:%d", ep.Address, ep.Port)
	}

	in := prompt.Input(">>> ", completer,
		prompt.OptionTitle("fleetctl"),
		prompt.OptionPrefixTextColor(prompt.Yellow),
		prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
		prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
		prompt.OptionSuggestionBGColor(prompt.DarkGray))
	fmt.Println("Your input: " + in)

}

func completer(in prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "users", Description: "Store the username and age"},
		{Text: "articles", Description: "Store the article text posted by user"},
		{Text: "comments", Description: "Store the text commented to articles"},
		{Text: "groups", Description: "Combine users with specific rules"},
	}
	return prompt.FilterHasPrefix(s, in.GetWordBeforeCursor(), true)
}
