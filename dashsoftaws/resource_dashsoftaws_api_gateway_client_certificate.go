package dashsoftaws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsApiGatewayClientCertificate() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsApiGatewayClientCertificateCreate,
		Read:   resourceDashsoftAwsApiGatewayClientCertificateRead,
		Update: resourceDashsoftAwsApiGatewayClientCertificateUpdate,
		Delete: resourceDashsoftAwsApiGatewayClientCertificateDelete,

		Schema: map[string]*schema.Schema{
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"certificatebody": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceDashsoftAwsApiGatewayClientCertificateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Creating API Gateway Client Certificate")

	input := &apigateway.GenerateClientCertificateInput{}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	out, err := conn.GenerateClientCertificate(input)
	if err != nil {
		return fmt.Errorf("Error generating API Gateway Client Certificate: %s", err)
	}
	log.Printf("[DEBUG] API Gateway Client Certificate %s generated", *out.ClientCertificateId)

	d.SetId(*out.ClientCertificateId)
	resourceDashsoftAwsApiGatewayClientCertificateRead(d, meta)

	return nil
}

func resourceDashsoftAwsApiGatewayClientCertificateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Client Certificate ID %s", d.Id())
	out, err := conn.GetClientCertificate(&apigateway.GetClientCertificateInput{
		ClientCertificateId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Client Certificate with description: %s", out.Description)

	d.SetId(*out.ClientCertificateId)
	if out.Description != nil {
		d.Set("description", *out.Description)
	}
	d.Set("certificatebody", *out.PemEncodedCertificate)
	return nil
}

func resourceDashsoftAwsApiGatewayClientCertificateUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	var patchOperations []*apigateway.PatchOperation

	d.Partial(true)

	if d.HasChange("description") {
		d.SetPartial("description")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/description"),
			Value: aws.String(d.Get("description").(string)),
		})
	}

	if len(patchOperations) > 0 {
		resp, err := conn.UpdateClientCertificate(&apigateway.UpdateClientCertificateInput{
			ClientCertificateId: aws.String(d.Id()),
			PatchOperations:     patchOperations,
		})

		if err != nil {
			return fmt.Errorf("Error updating ClientCertificate %s", err)
		}

		d.SetId(*resp.ClientCertificateId)
	}

	d.Partial(false)
	return resourceDashsoftAwsApiGatewayClientCertificateRead(d, meta)
}

func resourceDashsoftAwsApiGatewayClientCertificateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Deleting API Gateway Client Certificate %s", d.Id())

	ClientCertificateId := d.Id()
	_, err := conn.DeleteClientCertificate(&apigateway.DeleteClientCertificateInput{
		ClientCertificateId: aws.String(ClientCertificateId),
	})
	if err != nil {
		return fmt.Errorf("Error deleting API Gateway Client Certificate: %s", err)
	}
	log.Println("[INFO] API Gateway Client Certificate deleted")

	d.SetId("")

	log.Printf("[DEBUG] Deleted API Gateway Client Certificate %s", ClientCertificateId)
	return nil
}
