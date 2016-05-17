package dashsoftaws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDashsoftAwsIamGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsIamGroupCreate,
		Read:   resourceDashsoftAwsIamGroupRead,
		Update: resourceDashsoftAwsIamGroupUpdate,
		Delete: resourceDashsoftAwsIamGroupDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"unique_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
			},
		},
	}
}

func resourceDashsoftAwsIamGroupCreate(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)
	path := d.Get("path").(string)

	request := &iam.CreateGroupInput{
		Path:      aws.String(path),
		GroupName: aws.String(name),
	}

	createResp, err := iamconn.CreateGroup(request)
	if err != nil {
		return fmt.Errorf("Error creating IAM Group %s: %s", name, err)
	}
	return resourceDashsoftAwsIamGroupReadResult(d, createResp.Group)
}

func resourceDashsoftAwsIamGroupRead(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn
	name := d.Get("name").(string)

	request := &iam.GetGroupInput{
		GroupName: aws.String(name),
	}

	getResp, err := iamconn.GetGroup(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading IAM Group %s: %s", d.Id(), err)
	}
	return resourceDashsoftAwsIamGroupReadResult(d, getResp.Group)
}

func resourceDashsoftAwsIamGroupReadResult(d *schema.ResourceData, group *iam.Group) error {
	d.SetId(*group.GroupName)
	if err := d.Set("name", group.GroupName); err != nil {
		return err
	}
	if err := d.Set("arn", group.Arn); err != nil {
		return err
	}
	if err := d.Set("path", group.Path); err != nil {
		return err
	}
	if err := d.Set("unique_id", group.GroupId); err != nil {
		return err
	}
	return nil
}

func resourceDashsoftAwsIamGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("name") || d.HasChange("path") {
		iamconn := meta.(*AWSClient).iamconn
		on, nn := d.GetChange("name")
		_, np := d.GetChange("path")

		request := &iam.UpdateGroupInput{
			GroupName:    aws.String(on.(string)),
			NewGroupName: aws.String(nn.(string)),
			NewPath:      aws.String(np.(string)),
		}
		_, err := iamconn.UpdateGroup(request)
		if err != nil {
			return fmt.Errorf("Error updating IAM Group %s: %s", d.Id(), err)
		}
		return resourceDashsoftAwsIamGroupRead(d, meta)
	}
	return nil
}

func resourceDashsoftAwsIamGroupDelete(d *schema.ResourceData, meta interface{}) error {
	iamconn := meta.(*AWSClient).iamconn

	groupResp, groupErr := iamconn.GetGroup(&iam.GetGroupInput{
		GroupName: aws.String(d.Id()),
	})
	if groupErr != nil {
		for _, user := range groupResp.Users {
			removeUserInput := &iam.RemoveUserFromGroupInput{
				UserName:  aws.String(*user.UserName),
				GroupName: aws.String(d.Id()),
			}
			if _, removeUserErr := iamconn.RemoveUserFromGroup(removeUserInput); removeUserErr != nil {
				return fmt.Errorf("Error removing user %s from group %s: %s", d.Id(), user.UserName, removeUserErr)
			}
		}
	}

	request := &iam.DeleteGroupInput{
		GroupName: aws.String(d.Id()),
	}

	if _, err := iamconn.DeleteGroup(request); err != nil {
		return fmt.Errorf("Error deleting IAM Group %s: %s", d.Id(), err)
	}
	return nil
}
