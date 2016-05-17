package dashsoftaws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// Number of times to retry if a throttling- or test message exception occurs
const CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES = 30

// How long to sleep when a throttle-event happens
const CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_THROTTLE_SLEEP_MILLISECONDS = 2000

func resourceDashsoftAwsCloudwatchLogSubscriptionFilter() *schema.Resource {
	return &schema.Resource{
		Create: resourceDashsoftAwsCloudwatchLogSubscriptionFilterCreate,
		Read:   resourceDashsoftAwsCloudwatchLogSubscriptionFilterRead,
		Update: resourceDashsoftAwsCloudwatchLogSubscriptionFilterUpdate,
		Delete: resourceDashsoftAwsCloudwatchLogSubscriptionFilterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filter_pattern": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"log_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceDashsoftAwsCloudwatchLogSubscriptionFilterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	name := d.Get("name").(string)

	log_group := d.Get("log_group_name").(string)

	destination := d.Get("destination_arn").(string)
	if strings.HasPrefix(destination, "arn:aws:kinesis:") {
		destination_arn_sliced := strings.Split(destination, "/")
		destination_name := destination_arn_sliced[len(destination_arn_sliced)-1]

		kinesis_conn := meta.(*AWSClient).kinesisconn
		waitForKinesisStreamToActivate(kinesis_conn, destination_name)
	}

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)
	sleep_randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	log.Printf("[DEBUG] Creating SubscriptionFilter %#v", params)

	attemptCount := 1
	for attemptCount <= CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES {
		// Since the add permissions have a tendency to fail, we put the code in side.
		if strings.HasPrefix(destination, "arn:aws:lambda") {
			err := addPermissionsToLambdaFunction(d, meta)
			if err != nil {
				return err
			}
		}

		_, err := conn.PutSubscriptionFilter(&params)
		attemptCount += 1
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InvalidParameterException" {
					log.Printf("[DEBUG] Caught message: \"%s\", code: \"%s\". Attempt %d/%d: Sleeping for a bit to throttle back put request",
						awsErr.Message(), awsErr.Code(), attemptCount-1, CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_MAX_THROTTLE_RETRIES)
					// random delay 100-200% of THROTTLE_SLEEP
					time.Sleep(time.Duration(CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_THROTTLE_SLEEP_MILLISECONDS+sleep_randomizer.Intn(CLOUDWATCH_LOG_SUBSCRIPTION_FILTER_THROTTLE_SLEEP_MILLISECONDS/2)) * time.Millisecond)
				} else {
					// Some other non-retryable exception occurred
					return fmt.Errorf("[WARN] Error creating SubscriptionFilter (%s) for LogGroup (%s) to destination (%s), message: \"%s\", code: \"%s\"",
						name, log_group, destination, awsErr.Message(), awsErr.Code())
				}
			} else {
				// Non-AWS exception occurred, give up
				return fmt.Errorf("Error creating Cloudwatch logs subscription filter %s: %#v", name, err)
			}
		} else {
			d.SetId(cloudwatchLogSubscriptionFilterId(d.Get("log_group_name").(string)))
			return resourceDashsoftAwsCloudwatchLogSubscriptionFilterRead(d, meta)
		}
	}

	// Too many throttling events occurred, give up
	return fmt.Errorf("Unable to create Cloudwatch logs subscription filter '%s' after %d attempts", name, attemptCount)
}

func addPermissionsToLambdaFunction(d *schema.ResourceData, meta interface{}) error {
	lambda_conn := meta.(*AWSClient).lambdaconn

	name := d.Get("name").(string)
	log_group := d.Get("log_group_name").(string)
	destination := d.Get("destination_arn").(string)

	lambda_arn_sliced := strings.Split(destination, ":")
	function_name := lambda_arn_sliced[len(lambda_arn_sliced)-1]
	statement_id, err := lambdaPermissionStatementId(log_group, function_name)

	if err != nil {
		return err
	}

	if !permissionExists(function_name, statement_id, lambda_conn) {
		region := lambda_arn_sliced[3]
		accountid := lambda_arn_sliced[4]
		principal := fmt.Sprintf("logs.%s.amazonaws.com", region)
		source_arn := fmt.Sprintf("arn:aws:logs:%s:%s:log-group:%s:*", region, accountid, log_group)

		params := &lambda.AddPermissionInput{
			Action:        aws.String("lambda:InvokeFunction"),
			FunctionName:  aws.String(function_name),
			Principal:     aws.String(principal),
			StatementId:   aws.String(statement_id),
			SourceArn:     aws.String(source_arn),
			SourceAccount: aws.String(accountid),
		}

		log.Printf("[DEBUG] Attempting: to do add-access with params \"%#v\"", params)
		_, err := lambda_conn.AddPermission(params)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ResourceConflictException" {
					log.Printf("[DEBUG] Got a ResourceConflictException, but that is ok. Function=%s, log_group=%s", function_name, log_group)
				} else {
					return fmt.Errorf("[WARN] Error doing add-access for LogGroup (%s) to lambda (%s), message: \"%s\", code: \"%s\"",
						log_group, destination, awsErr.Message(), awsErr.Code())
				}
			} else {
				return fmt.Errorf("Error creating Cloudwatch logs subscription filter %s: %#v", name, err)
			}
		}
	}
	return nil
}

func resourceDashsoftAwsCloudwatchLogSubscriptionFilterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	params := getAwsCloudWatchLogsSubscriptionFilterInput(d)

	log.Printf("[DEBUG] Update SubscriptionFilter %#v", params)
	_, err := conn.PutSubscriptionFilter(&params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error updating SubscriptionFilter (%s) for LogGroup (%s), message: \"%s\", code: \"%s\"",
				d.Get("name").(string), d.Get("log_group_name").(string), awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(cloudwatchLogSubscriptionFilterId(d.Get("log_group_name").(string)))
	return resourceDashsoftAwsCloudwatchLogSubscriptionFilterRead(d, meta)
}

func getAwsCloudWatchLogsSubscriptionFilterInput(d *schema.ResourceData) cloudwatchlogs.PutSubscriptionFilterInput {
	name := d.Get("name").(string)
	destination_arn := d.Get("destination_arn").(string)
	filter_pattern := d.Get("filter_pattern").(string)
	log_group_name := d.Get("log_group_name").(string)

	params := cloudwatchlogs.PutSubscriptionFilterInput{
		FilterName:     aws.String(name),
		DestinationArn: aws.String(destination_arn),
		FilterPattern:  aws.String(filter_pattern),
		LogGroupName:   aws.String(log_group_name),
	}

	if _, ok := d.GetOk("role_arn"); ok {
		params.RoleArn = aws.String(d.Get("role_arn").(string))
	}

	return params
}

func resourceDashsoftAwsCloudwatchLogSubscriptionFilterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group := d.Get("log_group_name").(string)
	name := d.Get("name").(string) // "name" is a required field in the schema

	req := &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName:     aws.String(log_group),
		FilterNamePrefix: aws.String(name),
	}

	resp, err := conn.DescribeSubscriptionFilters(req)
	if err != nil {
		return fmt.Errorf("Error reading SubscriptionFilters for log group %s with name prefix %s: %#v", log_group, d.Get("name").(string), err)
	}

	for _, subscriptionFilter := range resp.SubscriptionFilters {
		if *subscriptionFilter.LogGroupName == log_group {
			d.SetId(cloudwatchLogSubscriptionFilterId(log_group))
			return nil // OK, matching subscription filter found
		}
	}

	return fmt.Errorf("Subscription filter for log group %s with name prefix %s not found!", log_group, d.Get("name").(string))
}

func resourceDashsoftAwsCloudwatchLogSubscriptionFilterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatchlogsconn

	log_group := d.Get("log_group_name").(string)
	name := d.Get("name").(string)
	destination := d.Get("destination_arn").(string)

	if strings.HasPrefix(destination, "arn:aws:lambda") {
		// access permissions should also be cleaned up
		lambda_conn := meta.(*AWSClient).lambdaconn

		lambda_arn_sliced := strings.Split(destination, ":")
		function_name := lambda_arn_sliced[len(lambda_arn_sliced)-1]
		statement_id, err := lambdaPermissionStatementId(log_group, function_name)

		if err != nil {
			return err
		}

		if permissionExists(function_name, statement_id, lambda_conn) {
			_, err := lambda_conn.RemovePermission(&lambda.RemovePermissionInput{
				FunctionName: aws.String(function_name),
				StatementId:  aws.String(statement_id),
			})
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					log.Printf("[WARN] Error removing the access permission SID (%s) for lambda function (%s), message: \"%s\", code: \"%s\"",
						statement_id, function_name, awsErr.Message(), awsErr.Code())
				} else {
					log.Printf("[WARN] Error removing the access permission from lambda function %s: %#v", function_name, err)
				}
			}
		}
	}

	params := &cloudwatchlogs.DeleteSubscriptionFilterInput{
		FilterName:   aws.String(name),      // Required
		LogGroupName: aws.String(log_group), // Required
	}
	_, err := conn.DeleteSubscriptionFilter(params)

	if err != nil {
		return fmt.Errorf(
			"Error deleting Subscription Filter from log group: %s with name filter name %s", log_group, name)
	}
	d.SetId("")
	return nil
}

func waitForKinesisStreamToActivate(conn *kinesis.Kinesis, stream_name string) error {
	// If destination is Kinesis stream, then it must be ACTIVE before creating SubscriptionFilter
	wait := resource.StateChangeConf{
		Pending:    []string{"CREATING", "UPDATING", "DELETING"},
		Target:     []string{"ACTIVE"},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Kinesis stream %s is ACTIVE", stream_name)
			resp, err := conn.DescribeStream(&kinesis.DescribeStreamInput{
				StreamName: aws.String(stream_name),
			})
			if err != nil {
				return resp, "FAILED", err
			}
			stream_status := *resp.StreamDescription.StreamStatus
			log.Printf("[DEBUG] Kinesis stream %s is %s checking for ACTIVE", stream_name, stream_status)
			return resp, stream_status, nil
		},
	}

	_, err := wait.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func permissionExists(function_name string, statementid string, lambda_conn *lambda.Lambda) bool {

	resp, err := lambda_conn.GetPolicy(&lambda.GetPolicyInput{
		FunctionName: aws.String(function_name),
	})

	type PolicyDocument struct {
		Version   string
		Statement []struct {
			Resource, Effect, Sid string
		}
	}

	if err != nil {
		log.Printf("[DEBUG] GetPolicy returns \"%#v\" - maybe no access permissions exists?", err)
		return false
	} else {
		dec := json.NewDecoder(strings.NewReader(*resp.Policy))
		for {
			var m PolicyDocument
			if err := dec.Decode(&m); err == io.EOF {
				break
			} else if err != nil {
				log.Printf("[DEBUG] Decoding access policy failed \"%#v\"", resp.Policy)
				log.Fatal(err)
			}

			for _, statement := range m.Statement {
				if statement.Sid == statementid {
					return true
				}
			}
		}
		log.Printf("[DEBUG] Statement Id \"%s\" not found in policy for function \"%s\"", statementid, function_name)
		return false
	}

}

func lambdaPermissionStatementId(log_group string, lambda string) (string, error) {
	// log_group chars not allowed in statementid: '/' (forward slash), and '.'

	var stmtid string = fmt.Sprintf("%s-%s", lambda, log_group)
	stmtid = strings.Replace(stmtid, "z", "z0", -1)
	stmtid = strings.Replace(stmtid, "/", "z1", -1)
	stmtid = strings.Replace(stmtid, ".", "z2", -1)

	if len(stmtid) > 100 {
		return "", fmt.Errorf("[Error] Could not create statementid, as it would be to long: log_group: %s, lambda: %s", log_group, lambda)
	} else {
		return stmtid, nil
	}
}

func cloudwatchLogSubscriptionFilterId(log_group_name string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("%s-", log_group_name)) // only one filter allowed per log_group_name at the moment

	return fmt.Sprintf("cwlsf-%d", hashcode.String(buf.String()))
}
