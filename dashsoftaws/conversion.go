package dashsoftaws

import (
	"github.com/aws/aws-sdk-go/aws"
)

func makeAwsStringList(in []interface{}) []*string {
	ret := make([]*string, len(in), len(in))
	for i := 0; i < len(in); i++ {
		ret[i] = aws.String(in[i].(string))
	}
	return ret
}
