package dashsoftaws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsApiGatewayBasePathMapping() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsApiGatewayBasePathMappingCreate,
		Read:   resourceDashsoftAwsApiGatewayBasePathMappingRead,
		Update: resourceDashsoftAwsApiGatewayBasePathMappingUpdate,
		Delete: resourceDashsoftAwsApiGatewayBasePathMappingDelete,

		Schema: map[string]*schema.Schema{
			"domainname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"restapiid": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"stage": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"basepath": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDashsoftAwsApiGatewayBasePathMappingCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName := d.Get("domainname").(string)
	restApiId := d.Get("restapiid").(string)

	log.Printf("[DEBUG] Creating API Gateway Base Path Mapping for Domain %s with RestApi Id %&s", domainName, restApiId)

	input := &apigateway.CreateBasePathMappingInput{
		DomainName: aws.String(domainName),
		RestApiId:  aws.String(restApiId),
	}

	if v, ok := d.GetOk("stage"); ok {
		input.Stage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("basepath"); ok {
		input.BasePath = aws.String(v.(string))
	}

	out, err := conn.CreateBasePathMapping(input)
	if err != nil {
		return fmt.Errorf("Error creating API Gateway Base Path Mapping: %s", err)
	}
	log.Printf("[DEBUG] API Gateway Base Path Mapping created")

	d.SetId(fmt.Sprintf("%s:%s", domainName, *out.BasePath))
	return nil
}

func resourceDashsoftAwsApiGatewayBasePathMappingRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	domainName, basePath := resourceDashsoftAwsApiGatewayBasePathMappingParseId(d.Id())

	log.Printf("[DEBUG] Reading API Gateway Base Path Mapping ID %s", d.Id())
	out, err := conn.GetBasePathMapping(&apigateway.GetBasePathMappingInput{
		BasePath:   aws.String(basePath),
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Received API Gateway Base Path Mapping %s for domain %s", *out.BasePath, domainName)

	if v, ok := d.GetOk("basepath"); ok {
		d.Set("basepath", aws.String(v.(string)))
	} else {
		d.Set("basepath", aws.String(""))
	}

	d.SetId(fmt.Sprintf("%s:%s", domainName, *out.BasePath))
	return nil
}

func resourceDashsoftAwsApiGatewayBasePathMappingUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	originalBasePath, originalDomainName := resourceDashsoftAwsApiGatewayBasePathMappingParseId(d.Id())
	var patchOperations []*apigateway.PatchOperation

	d.Partial(true)

	if d.HasChange("basepath") {
		d.SetPartial("basepath")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/basePath"),
			Value: aws.String(d.Get("basepath").(string)),
		})
	}

	if d.HasChange("restapiid") {
		d.SetPartial("restapiid")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/restapiId"),
			Value: aws.String(d.Get("restapiid").(string)),
		})
	}

	if d.HasChange("stage") {
		d.SetPartial("stage")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/stage"),
			Value: aws.String(d.Get("stage").(string)),
		})
	}

	if len(patchOperations) > 0 {
		resp, err := conn.UpdateBasePathMapping(&apigateway.UpdateBasePathMappingInput{
			BasePath:        &originalBasePath,
			DomainName:      &originalDomainName,
			PatchOperations: patchOperations,
		})

		if err != nil {
			return fmt.Errorf("Error updating BasePathMapping %s", err)
		}

		d.SetId(fmt.Sprintf("%s:%s", originalDomainName, *resp.BasePath))
	}

	d.Partial(false)
	return resourceDashsoftAwsApiGatewayBasePathMappingRead(d, meta)
}

func resourceDashsoftAwsApiGatewayBasePathMappingDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Deleting API Gateway Base Path Mapping %s", d.Id())

	domainName, basePath := resourceDashsoftAwsApiGatewayBasePathMappingParseId(d.Id())
	_, err := conn.DeleteBasePathMapping(&apigateway.DeleteBasePathMappingInput{
		BasePath:   aws.String(basePath),
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return fmt.Errorf("Error deleting API Gateway Base Path Mapping: %s", err)
	}
	log.Println("[INFO] API Gateway Base Path Mapping deleted")

	d.SetId("")

	log.Printf("[DEBUG] Deleted API Gateway Base Path Mapping %s for domain %s", basePath, domainName)
	return nil
}

func resourceDashsoftAwsApiGatewayBasePathMappingParseId(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
