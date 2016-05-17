package dashsoftaws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsApiGatewayDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsApiGatewayDeploymentCreate,
		Read:   resourceDashsoftAwsApiGatewayDeploymentRead,
		Update: resourceDashsoftAwsApiGatewayDeploymentUpdate,
		Delete: resourceDashsoftAwsApiGatewayDeploymentDelete,

		Schema: map[string]*schema.Schema{
			"restapiid": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cacheclusterenabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"cacheclustersize": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"stage_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"stagedescription": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"variables": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"clientcertificateid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"cloudwatchlogsloglevel": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "OFF",
			},
			"datatrace": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"burstlimit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ratelimit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceDashsoftAwsApiGatewayDeploymentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	restApiId := d.Get("restapiid").(string)
	stageName := d.Get("stage_name").(string)

	input := &apigateway.CreateDeploymentInput{
		RestApiId: aws.String(restApiId),
		StageName: aws.String(stageName),
	}

	if v, ok := d.GetOk("cacheclusterenabled"); ok {
		input.CacheClusterEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("cacheclustersize"); ok {
		input.CacheClusterSize = aws.String(v.(string))
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("stagedescription"); ok {
		input.StageDescription = aws.String(v.(string))
	}

	if v, ok := d.GetOk("variables"); ok {
		input.Variables = stringMapToPointers(v.(map[string]interface{}))
	}

	log.Printf("[DEBUG] Creating API Gateway Deployment with Stage %s", stageName)

	deployment, err := conn.CreateDeployment(input)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] API Gateway Deployment %s created", *deployment.Id)

	var patchOperations []*apigateway.PatchOperation

	if v, ok := d.GetOk("clientcertificateid"); ok {
		clientCertificateId := v.(string)
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/clientCertificateId"),
			Value: aws.String(clientCertificateId),
		})
	}

	if v, ok := d.GetOk("cloudwatchlogsloglevel"); ok {
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/logging/loglevel"),
			Value: aws.String(v.(string)),
		})
	}

	if v, ok := d.GetOk("datatrace"); ok {
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/logging/dataTrace"),
			Value: aws.String(fmt.Sprintf("%t", v.(bool))),
		})
	}

	if v, ok := d.GetOk("burstlimit"); ok {
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/throttling/burstLimit"),
			Value: aws.String(fmt.Sprintf("%d", v.(int))),
		})
	}

	if v, ok := d.GetOk("ratelimit"); ok {
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/throttling/rateLimit"),
			Value: aws.String(fmt.Sprintf("%d", v.(int))),
		})
	}

	if len(patchOperations) > 0 {
		conn.UpdateStage(&apigateway.UpdateStageInput{
			RestApiId:       aws.String(restApiId),
			StageName:       aws.String(stageName),
			PatchOperations: patchOperations,
		})

		if err != nil {
			return fmt.Errorf("Error updating Stage %s: %s", stageName, err)
		}
	}

	d.SetId(*deployment.Id)

	resourceDashsoftAwsApiGatewayDeploymentRead(d, meta)
	return nil
}

func resourceDashsoftAwsApiGatewayDeploymentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	log.Printf("[DEBUG] Reading API Gateway Deployment ID %s", d.Id())

	out, err := conn.GetDeployment(&apigateway.GetDeploymentInput{
		DeploymentId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("restapiid").(string)),
	})

	if err != nil {
		return err
	}

	if out.Description != nil {
		d.Set("description", *out.Description)
	}

	d.SetId(*out.Id)
	return nil
}

func resourceDashsoftAwsApiGatewayDeploymentUpdate(d *schema.ResourceData, meta interface{}) error {
	//this update method doesn't work as it should
	//The stages should probarly be split into its own resource, that then relies on the api deployment
	conn := meta.(*AWSClient).apigateway

	var patchOperations []*apigateway.PatchOperation

	d.Partial(true)

	if d.HasChange("burstlimit") {
		d.SetPartial("burstlimit")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/throttling/burstLimit"),
			Value: aws.String(d.Get("burstlimit").(string)),
		})
	}

	if d.HasChange("ratelimit") {
		d.SetPartial("ratelimit")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/throttling/rateLimit"),
			Value: aws.String(d.Get("ratelimit").(string)),
		})
	}

	if d.HasChange("cloudwatchlogsloglevel") {
		d.SetPartial("cloudwatchlogsloglevel")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/logging/loglevel"),
			Value: aws.String(d.Get("cloudwatchlogsloglevel").(string)),
		})
	}

	if d.HasChange("datatrace") {
		d.SetPartial("datatrace")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/*/*/logging/dataTrace"),
			Value: aws.String(d.Get("datatrace").(string)),
		})
	}

	if d.HasChange("cacheclusterenabled") {
		d.SetPartial("cacheclusterenabled")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/CacheClusterEnabled"),
			Value: aws.String(d.Get("cacheclusterenabled").(string)),
		})
	}

	if d.HasChange("cacheclustersize") {
		d.SetPartial("cacheclustersize")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/CacheClusterSize"),
			Value: aws.String(d.Get("cacheclusterenabled").(string)),
		})
	}

	if d.HasChange("clientcertificateid") {
		d.SetPartial("clientcertificateid")
		patchOperations = append(patchOperations, &apigateway.PatchOperation{
			Op:    aws.String(apigateway.OpReplace),
			Path:  aws.String("/clientCertificateId"),
			Value: aws.String(d.Get("clientcertificateid").(string)),
		})
	}

	if len(patchOperations) > 0 {
		resp, err := conn.UpdateDeployment(&apigateway.UpdateDeploymentInput{
			DeploymentId:    aws.String(d.Id()),
			PatchOperations: patchOperations,
		})

		if err != nil {
			return fmt.Errorf("Error updating deployment %s", err)
		}

		d.SetId(*resp.Id)
	}

	d.Partial(false)
	return resourceDashsoftAwsApiGatewayDeploymentRead(d, meta)
}

func resourceDashsoftAwsApiGatewayDeploymentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).apigateway

	restApiId := d.Get("restapiid").(string)

	if v, ok := d.GetOk("stage_name"); ok {
		stageName := v.(string)
		log.Printf("[DEBUG] Delete stage with name %s (if it still exists)", stageName)
		_, err := conn.DeleteStage(&apigateway.DeleteStageInput{
			RestApiId: aws.String(restApiId),
			StageName: aws.String(stageName),
		})
		if err != nil {
			log.Printf("[INFO] Ignored error when deleting stage %s", err)
		}
	}

	log.Printf("[DEBUG] Deleting API Gateway Deployment %s", d.Id())
	_, err := conn.DeleteDeployment(&apigateway.DeleteDeploymentInput{
		DeploymentId: aws.String(d.Id()),
		RestApiId:    aws.String(d.Get("restapiid").(string)),
	})
	if err != nil {
		return fmt.Errorf("Error deleting API Gateway Deployment: %s", err)
	}
	log.Println("[INFO] API Gateway Deployment deleted")

	d.SetId("")

	log.Printf("[DEBUG] Deleted API Gateway Deployment %s", restApiId)
	return nil
}
