package dashsoftaws

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	// The actual provider
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["secret_key"],
			},

			"profile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["profile"],
			},

			"shared_credentials_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["shared_credentials_file"],
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["token"],
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},

			"max_retries": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     11,
				Description: descriptions["max_retries"],
			},

			"allowed_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set:           schema.HashString,
			},

			"forbidden_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set:           schema.HashString,
			},

			"dynamodb_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["dynamodb_endpoint"],
			},

			"kinesis_endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["kinesis_endpoint"],
			},
			"endpoints": endpointsSchema(),

			"insecure": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["insecure"],
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"dashsoftaws_api_gateway_base_path_mapping":      resourceDashsoftAwsApiGatewayBasePathMapping(),
			"dashsoftaws_api_gateway_client_certificate":     resourceDashsoftAwsApiGatewayClientCertificate(),
			"dashsoftaws_api_gateway_deployment":             resourceDashsoftAwsApiGatewayDeployment(),
			"dashsoftaws_api_gateway_domain_name":            resourceDashsoftAwsApiGatewayDomainName(),
			"dashsoftaws_dynamodb_table":                     resourceDashsoftAwsDynamodbTable(),
			"dashsoftaws_iam_group":                          resourceDashsoftAwsDynamodbTable(),
			"dashsoftaws_kms_grant":                          resourceDashsoftAwsKMSGrant(),
			"dashsoftaws_cloudwatch_log_subscription_filter": resourceDashsoftAwsCloudwatchLogSubscriptionFilter(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"region": "The region where AWS operations will take place. Examples\n" +
			"are us-east-1, us-west-2, etc.",

		"access_key": "The access key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"secret_key": "The secret key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"profile": "The profile for API operations. If not set, the default profile\n" +
			"created with `aws configure` will be used.",

		"shared_credentials_file": "The path to the shared credentials file. If not set\n" +
			"this defaults to ~/.aws/credentials.",

		"token": "session token. A session token is only required if you are\n" +
			"using temporary security credentials.",

		"max_retries": "The maximum number of times an AWS API request is\n" +
			"being executed. If the API request still fails, an error is\n" +
			"thrown.",

		"dynamodb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to dynamodb-local.",

		"kinesis_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n" +
			"It's typically used to connect to kinesalite.",

		"iam_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"ec2_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"elb_endpoint": "Use this to override the default endpoint URL constructed from the `region`.\n",

		"insecure": "Explicitly allow the provider to perform \"insecure\" SSL requests. If omitted," +
			"default value is `false`",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey:        d.Get("access_key").(string),
		SecretKey:        d.Get("secret_key").(string),
		Profile:          d.Get("profile").(string),
		CredsFilename:    d.Get("shared_credentials_file").(string),
		Token:            d.Get("token").(string),
		Region:           d.Get("region").(string),
		MaxRetries:       d.Get("max_retries").(int),
		DynamoDBEndpoint: d.Get("dynamodb_endpoint").(string),
		KinesisEndpoint:  d.Get("kinesis_endpoint").(string),
		Insecure:         d.Get("insecure").(bool),
	}

	endpointsSet := d.Get("endpoints").(*schema.Set)

	for _, endpointsSetI := range endpointsSet.List() {
		endpoints := endpointsSetI.(map[string]interface{})
		config.IamEndpoint = endpoints["iam"].(string)
		config.Ec2Endpoint = endpoints["ec2"].(string)
		config.ElbEndpoint = endpoints["elb"].(string)
	}

	if v, ok := d.GetOk("allowed_account_ids"); ok {
		config.AllowedAccountIds = v.(*schema.Set).List()
	}

	if v, ok := d.GetOk("forbidden_account_ids"); ok {
		config.ForbiddenAccountIds = v.(*schema.Set).List()
	}

	return config.Client()
}

// This is a global MutexKV for use within this plugin.
var awsMutexKV = mutexkv.NewMutexKV()

func endpointsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"iam": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["iam_endpoint"],
				},

				"ec2": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["ec2_endpoint"],
				},

				"elb": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "",
					Description: descriptions["elb_endpoint"],
				},
			},
		},
		Set: endpointsToHash,
	}
}

func endpointsToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["iam"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["ec2"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["elb"].(string)))

	return hashcode.String(buf.String())
}
