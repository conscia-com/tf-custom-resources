package dashsoftaws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsApiGatewayDomainName() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsApiGatewayDomainNameCreate,
		Read:   resourceDashsoftAwsApiGatewayDomainNameRead,
		Delete: resourceDashsoftAwsApiGatewayDomainNameDelete,

		Schema: map[string]*schema.Schema{
			"domainname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificatename": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificateprivatekey": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificatebody": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"certificatechain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"distributiondomainname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDashsoftAwsApiGatewayDomainNameCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName := d.Get("domainname").(string)
	log.Printf("[DEBUG] Creating API Gateway Domain Name %s", domainName)

	out, err := conn.CreateDomainName(&apigateway.CreateDomainNameInput{
		CertificateChain:      aws.String(d.Get("certificatechain").(string)),
		CertificateBody:       aws.String(d.Get("certificatebody").(string)),
		CertificateName:       aws.String(d.Get("certificatename").(string)),
		CertificatePrivateKey: aws.String(d.Get("certificateprivatekey").(string)),
		DomainName:            aws.String(domainName),
	})
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Domain Name: %s", err)
	}
	log.Printf("[DEBUG] API Gateway Domain Name %s created with DistributionDomainName %s", *out.DomainName, *out.DistributionDomainName)

	d.SetId(*out.DomainName)
	resourceDashsoftAwsApiGatewayDomainNameRead(d, meta)

	return nil
}

func resourceDashsoftAwsApiGatewayDomainNameRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Domain Name ID %s", d.Id())
	out, err := conn.GetDomainName(&apigateway.GetDomainNameInput{
		DomainName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error reading API Gateway Domain Name: %s", err)
	}
	log.Printf("[DEBUG] API Gateway Domain Name %s created with DistributionDomainName %s", *out.DomainName, *out.DistributionDomainName)

	d.SetId(*out.DomainName)
	d.Set("distributiondomainname", *out.DistributionDomainName)
	return nil
}

func resourceDashsoftAwsApiGatewayDomainNameDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Deleting API Gateway Domain Name %s", d.Id())

	DomainNameId := d.Id()
	_, err := conn.DeleteDomainName(&apigateway.DeleteDomainNameInput{
		DomainName: aws.String(d.Id()),
	})
	if err != nil {
		return fmt.Errorf("Error deleting API Gateway Domain Name: %s", err)
	}
	log.Println("[INFO] API Gateway Domain Name deleted")
	d.SetId("")

	log.Printf("[DEBUG] Deleted API Gateway Domain Name %s", DomainNameId)
	return nil
}
