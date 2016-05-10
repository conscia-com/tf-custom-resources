package dashsoftaws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsKMSGrant() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsKMSGrantCreate,
		Read:   resourceDashsoftAwsKMSGrantRead,
		//		Update: resourceDashsoftAwsKMSGrantUpdate,
		Delete: resourceDashsoftAwsKMSGrantDelete,

		Schema: map[string]*schema.Schema{
			"granteeprincipal": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"keyid": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"constraints": &schema.Schema{
				Type:     schema.TypeSet,
				ForceNew: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"encryptioncontextequals": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
						"encryptioncontextsubset": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			"granttokens": &schema.Schema{
				Type:     schema.TypeList,
				ForceNew: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"operations": &schema.Schema{
				Type:     schema.TypeList,
				ForceNew: true,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"retiringprincipal": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceDashsoftAwsKMSGrantCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	granteePrincipal := d.Get("granteeprincipal").(string)
	keyId := d.Get("keyid").(string)

	log.Printf("[DEBUG] Creating KMS Grant for key %s", keyId)

	input := &kms.CreateGrantInput{
		GranteePrincipal: aws.String(granteePrincipal),
		KeyId:            aws.String(keyId),
	}

	if v, ok := d.GetOk("constraints"); ok {
		input.Constraints = &kms.GrantConstraints{}
		log.Printf("[DEBUG] This is how constraints look %#v", v)
	}

	if v, ok := d.GetOk("granttokens"); ok {
		input.GrantTokens = makeAwsStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("operations"); ok {
		input.Operations = makeAwsStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("name"); ok {
		input.Name = aws.String(v.(string))
	}

	if v, ok := d.GetOk("retiringprincipal"); ok {
		input.RetiringPrincipal = aws.String(v.(string))
	}

	out, err := conn.CreateGrant(input)
	if err != nil {
		return fmt.Errorf("Error creating KMS Grant: %s", err)
	}
	log.Printf("[DEBUG] KMS Grant created")

	d.SetId(*out.GrantId)
	d.Set("token", *out.GrantToken)
	return nil
}

func resourceDashsoftAwsKMSGrantRead(d *schema.ResourceData, meta interface{}) error {
	//	conn := meta.(*AWSClient).kmsconn
	//
	//	domainName, basePath := resourceDashsoftAwsKMSGrantParseId(d.Id())
	//
	//	log.Printf("[DEBUG] Reading KMS Grant ID %s", d.Id())
	//	out, err := conn.ListGrants(&kms.ListGrantsInput{
	//		BasePath:   aws.String(basePath),
	//		DomainName: aws.String(domainName),
	//	})
	//	if err != nil {
	//		return err
	//	}
	//	log.Printf("[DEBUG] Received KMS Grant %s for domain %s", *out.BasePath, domainName)
	//
	//	if v, ok := d.GetOk("basepath"); ok {
	//		d.Set("basepath", aws.String(v.(string)))
	//	} else {
	//		d.Set("basepath", aws.String(""))
	//	}
	//
	//	d.SetId(fmt.Sprintf("%s:%s", domainName, *out.BasePath))
	return nil
}

//func resourceDashsoftAwsKMSGrantUpdate(d *schema.ResourceData, meta interface{}) error {
//	conn := meta.(*AWSClient).kmsconn
//
//	originalBasePath, originalDomainName := resourceDashsoftAwsKMSGrantParseId(d.Id())
//	var patchOperations []*KMS.PatchOperation
//
//	d.Partial(true)
//
//	if d.HasChange("basepath") {
//		d.SetPartial("basepath")
//		patchOperations = append(patchOperations, &KMS.PatchOperation{
//			Op:    aws.String(KMS.OpReplace),
//			Path:  aws.String("/basePath"),
//			Value: aws.String(d.Get("basepath").(string)),
//		})
//	}
//
//	if d.HasChange("restapiid") {
//		d.SetPartial("restapiid")
//		patchOperations = append(patchOperations, &KMS.PatchOperation{
//			Op:    aws.String(KMS.OpReplace),
//			Path:  aws.String("/restapiId"),
//			Value: aws.String(d.Get("restapiid").(string)),
//		})
//	}
//
//	if d.HasChange("stage") {
//		d.SetPartial("stage")
//		patchOperations = append(patchOperations, &KMS.PatchOperation{
//			Op:    aws.String(KMS.OpReplace),
//			Path:  aws.String("/stage"),
//			Value: aws.String(d.Get("stage").(string)),
//		})
//	}
//
//	if len(patchOperations) > 0 {
//		resp, err := conn.UpdateGrant(&KMS.UpdateGrantInput{
//			BasePath: &originalBasePath,
//			DomainName: &originalDomainName,
//			PatchOperations: patchOperations,
//		})
//
//		if err != nil {
//			return fmt.Errorf("Error updating Grant %s", err)
//		}
//
//		d.SetId(fmt.Sprintf("%s:%s", originalDomainName, *resp.BasePath))
//	}
//
//	d.Partial(false)
//	return resourceDashsoftAwsKMSGrantRead(d, meta)
//}

func resourceDashsoftAwsKMSGrantDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kmsconn

	grantId := d.Id()
	keyId := d.Get("keyid").(string)

	log.Printf("[DEBUG] Revoke KMS Grant %s", d.Id())

	_, err := conn.RevokeGrant(&kms.RevokeGrantInput{
		KeyId:   aws.String(keyId),
		GrantId: aws.String(grantId),
	})
	if err != nil {
		return fmt.Errorf("Error revoked KMS Grant: %s", err)
	}
	log.Println("[INFO] KMS Grant revoked")

	d.SetId("")

	log.Printf("[DEBUG] Revoked KMS Grant %s for key %s", grantId, keyId)
	return nil
}
