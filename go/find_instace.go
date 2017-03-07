package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/urfave/cli"
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getSession() *session.Session {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("ap-southeast-1")})
	check(err)
	return sess
}

func searchInstance(elbs *elb.DescribeLoadBalancersOutput, iid string) ([]string, bool) {
	var lbname []string
	found := false
	for _, l := range elbs.LoadBalancerDescriptions {
		for _, i := range l.Instances {
			if *i.InstanceId == iid {
				lbname = append(lbname, *l.LoadBalancerName)
				found = true
			}
		}
	}
	return lbname, found
}

var lbNames []string

func elbInfo(marker string, firstCall bool, iid string) []string {
	var params *elb.DescribeLoadBalancersInput

	sess := getSession()
	svc := elb.New(sess)

	if len(marker) == 0 && !firstCall {
		return lbNames
	}

	if len(marker) == 0 {
		params = &elb.DescribeLoadBalancersInput{}
	} else {
		params = &elb.DescribeLoadBalancersInput{
			Marker: aws.String(marker),
		}
	}

	resp, err := svc.DescribeLoadBalancers(params)
	check(err)
	lbname, found := searchInstance(resp, iid)
	if found {
		lbNames = append(lbNames, lbname...)
	}

	if resp.NextMarker != nil {
		marker = *resp.NextMarker
	} else {
		marker = ""
	}
	return elbInfo(marker, false, iid)
}
func instanceID(svc *ec2.EC2, ipadd string) string {
	params := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("addresses.private-ip-address"),
				Values: []*string{
					aws.String(ipadd),
				},
			},
		},
	}

	resp, err := svc.DescribeNetworkInterfaces(params)
	check(err)
	return *resp.NetworkInterfaces[0].Attachment.InstanceId
}

func findInstance(ipaddr string) {
	sess := getSession()
	svc := ec2.New(sess)
	iid := instanceID(svc, ipaddr)
	ec2params := &ec2.DescribeInstancesInput{

		InstanceIds: []*string{
			aws.String(iid),
		},
	}
	instanceInfo, err := svc.DescribeInstances(ec2params)
	check(err)
	fmt.Println(instanceInfo.Reservations[0].Instances[0].Tags)
	lbNames = elbInfo("", true, iid)
	fmt.Println("\n ELBs Instance registered with : ", strings.Join(lbNames, ","))
}


func main() {
	var ipad string

	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "ipaddr,i",
			Usage:       "IP address of the instance",
			Destination: &ipad,
		},
	}

	app.Action = func(c *cli.Context) error {
		if len(ipad) > 0 {
			findInstance(ipad)
		}
		return nil
	}

	app.Run(os.Args)

}
