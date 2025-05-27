package common

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func GetWriteOnlyValue(d *schema.ResourceData, p cty.Path, t cty.Type) (cty.Value, error) {
	var empty cty.Value

	if d.GetRawConfig().IsNull() {
		return empty, errors.New("raw config is null")
	}

	value, di := d.GetRawConfigAt(p)
	if di.HasError() {
		return empty, errors.New("error retrieving write-only value")
	}

	if !value.Type().Equals(t) {
		return empty, fmt.Errorf("value is not a %s", t.FriendlyName())
	}

	return value, nil
}

func GetWriteOnlyStringValue(d *schema.ResourceData, p cty.Path) (string, error) {
	value, err := GetWriteOnlyValue(d, p, cty.String)
	if err != nil {
		return "", err
	}

	if !value.IsNull() {
		return value.AsString(), nil
	}

	return "", nil
}
