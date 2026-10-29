package spell

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Language is a dictionary mdedit can use.
type Language struct {
	Code string // "en" is the built-in English list; others are downloaded
	Name string
}

// BuiltIn is the code of the English word list compiled into mdedit.
const BuiltIn = "en"

// Languages lists the built-in dictionary and the ones that can be
// downloaded, by name.
var Languages = []Language{
	{BuiltIn, "English (built in)"},
	{"bg", "Bulgarian"}, {"br", "Breton"}, {"ca", "Catalan"}, {"hr", "Croatian"},
	{"cs", "Czech"}, {"da", "Danish"}, {"nl", "Dutch"},
	{"en-AU", "English (Australia)"}, {"en-CA", "English (Canada)"},
	{"en-GB", "English (United Kingdom)"}, {"en-ZA", "English (South Africa)"},
	{"en-US", "English (United States, Hunspell)"},
	{"eo", "Esperanto"}, {"et", "Estonian"}, {"fo", "Faroese"}, {"fr", "French"},
	{"fy", "Frisian"}, {"gl", "Galician"}, {"ka", "Georgian"}, {"de", "German (Germany)"},
	{"de-AT", "German (Austria)"}, {"de-CH", "German (Switzerland)"}, {"el", "Greek"},
	{"he", "Hebrew"}, {"hu", "Hungarian"}, {"is", "Icelandic"}, {"ia", "Interlingua"},
	{"ga", "Irish"}, {"it", "Italian"}, {"ko", "Korean"}, {"la", "Latin"},
	{"lv", "Latvian"}, {"lt", "Lithuanian"}, {"lb", "Luxembourgish"}, {"mk", "Macedonian"},
	{"mn", "Mongolian"}, {"nds", "Low German"}, {"ne", "Nepali"},
	{"nb", "Norwegian (Bokmål)"}, {"nn", "Norwegian (Nynorsk)"}, {"oc", "Occitan"},
	{"fa", "Persian"}, {"pl", "Polish"}, {"pt", "Portuguese (Brazil)"},
	{"pt-PT", "Portuguese (Portugal)"}, {"ro", "Romanian"}, {"ru", "Russian"},
	{"gd", "Scottish Gaelic"}, {"sr", "Serbian (Cyrillic)"}, {"sr-Latn", "Serbian (Latin)"},
	{"sk", "Slovak"}, {"sl", "Slovenian"}, {"es", "Spanish"}, {"es-MX", "Spanish (Mexico)"},
	{"sv", "Swedish"}, {"tr", "Turkish"}, {"uk", "Ukrainian"}, {"vi", "Vietnamese"},
	{"cy", "Welsh"},
}

// LanguageName returns the name of a language code, or the code itself.
func LanguageName(code string) string {
	for _, l := range Languages {
		if l.Code == code {
			return l.Name
		}
	}
	return code
}

// dictionarySource is where dictionaries are downloaded from: the
// LibreOffice and Hunspell dictionaries collected, converted to UTF-8 and
// licensed each under its own terms at github.com/wooorm/dictionaries.
const dictionarySource = "https://raw.githubusercontent.com/wooorm/dictionaries/main/dictionaries/"

// sourceDir maps a code to its folder in the source, where they differ.
func sourceDir(code string) string {
	if code == "en-US" {
		return "en"
	}
	return code
}

// Download fetches a language's .aff and .dic files.
func Download(code string) (aff, dic []byte, err error) {
	if aff, err = fetch(code, "index.aff"); err != nil {
		return nil, nil, err
	}
	if dic, err = fetch(code, "index.dic"); err != nil {
		return nil, nil, err
	}
	return aff, dic, nil
}

var httpClient = &http.Client{Timeout: 2 * time.Minute}

func fetch(code, file string) ([]byte, error) {
	resp, err := httpClient.Get(dictionarySource + sourceDir(code) + "/" + file)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading the %s dictionary: %s", LanguageName(code), resp.Status)
	}
	return io.ReadAll(resp.Body)
}
