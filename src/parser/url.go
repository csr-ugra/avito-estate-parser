package parser

import (
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/src/db"
)

func BuildUrl(location *db.EstateLocationModel, target *db.EstateTargetModel) (url string, err error) {
	const urlFormat = "https://www.avito.ru/%s/%s"

	if location.UrlPart == "" {
		return "", fmt.Errorf("location model does not have a url part")
	}

	if target.UrlPart == "" {
		return "", fmt.Errorf("target model does not have a url part")
	}

	url = fmt.Sprintf(urlFormat, location.UrlPart, target.UrlPart)

	return url, nil
}
