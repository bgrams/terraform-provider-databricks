package secrets

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/terraform-provider-databricks/common"
	"github.com/hashicorp/go-cty/cty"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func readSecret(ctx context.Context, w *databricks.WorkspaceClient, scope string, key string) (workspace.SecretMetadata, error) {
	var secretMeta workspace.SecretMetadata
	secrets := w.Secrets.ListSecrets(ctx, workspace.ListSecretsRequest{Scope: scope})
	for secrets.HasNext(ctx) {
		secret, err := secrets.Next(ctx)
		if err != nil {
			return secretMeta, err
		}
		if secret.Key == key {
			secretMeta.Key = secret.Key
			secretMeta.LastUpdatedTimestamp = secret.LastUpdatedTimestamp
			return secretMeta, nil
		}
	}

	return secretMeta, &apierr.APIError{
		ErrorCode:  "NOT_FOUND",
		StatusCode: 404,
		Message:    fmt.Sprintf("no secret Scope found with secret metadata scope name: %s and key: %s", scope, key),
	}
}

// ResourceSecret manages secrets
func ResourceSecret() common.Resource {
	p := common.NewPairSeparatedID("scope", "key", "|||")
	s := map[string]*schema.Schema{
		"string_value": {
			Type:          schema.TypeString,
			ValidateFunc:  validation.StringIsNotEmpty,
			ConflictsWith: []string{"string_value_wo"},
			ForceNew:      true,
			Sensitive:     true,
			Optional:      true,
		},
		"string_value_wo": {
			Type:          schema.TypeString,
			ConflictsWith: []string{"string_value"},
			RequiredWith:  []string{"string_value_wo_version"},
			Optional:      true,
			Sensitive:     true,
			WriteOnly:     true,
		},
		"string_value_wo_version": {
			Type:         schema.TypeInt,
			RequiredWith: []string{"string_value_wo"},
			Optional:     true,
		},
		"scope": {
			Type:         schema.TypeString,
			ValidateFunc: validScope,
			Required:     true,
			ForceNew:     true,
		},
		"key": {
			Type:         schema.TypeString,
			ValidateFunc: validScope,
			Required:     true,
			ForceNew:     true,
		},
		"last_updated_timestamp": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"config_reference": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}
	return common.Resource{
		Schema: s,
		Create: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			w, err := c.WorkspaceClient()
			if err != nil {
				return err
			}
			var putSecretReq workspace.PutSecret
			common.DataToStructPointer(d, s, &putSecretReq)

			stringValueWO, err := common.GetWriteOnlyStringValue(d, cty.GetAttrPath("string_value_wo"))
			if err != nil {
				return err
			}

			if stringValueWO != "" {
				putSecretReq.StringValue = stringValueWO
			}

			err = w.Secrets.PutSecret(ctx, putSecretReq)
			if err != nil {
				return err
			}
			p.Pack(d)
			return nil
		},
		Read: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			scope, key, err := p.Unpack(d)
			if err != nil {
				return err
			}
			w, err := c.WorkspaceClient()
			if err != nil {
				return err
			}
			m, err := readSecret(ctx, w, scope, key)
			if err != nil {
				return err
			}
			d.Set("config_reference", fmt.Sprintf("{{secrets/%s/%s}}", scope, key))
			return d.Set("last_updated_timestamp", m.LastUpdatedTimestamp)
		},
		Delete: func(ctx context.Context, d *schema.ResourceData, c *common.DatabricksClient) error {
			scope, key, err := p.Unpack(d)
			if err != nil {
				return err
			}
			w, err := c.WorkspaceClient()
			if err != nil {
				return err
			}
			return w.Secrets.DeleteSecret(ctx, workspace.DeleteSecret{
				Scope: scope,
				Key:   key,
			})
		},
		ValidateRawResourceConfigFuncs: []schema.ValidateRawResourceConfigFunc{
			validation.PreferWriteOnlyAttribute(cty.GetAttrPath("string_value"), cty.GetAttrPath("string_value_wo")),
		},
	}
}
