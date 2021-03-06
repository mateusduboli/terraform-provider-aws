package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSPolicyAttachment_basic(t *testing.T) {
	var out iam.ListEntitiesForPolicyOutput

	user1 := fmt.Sprintf("test-user-%d", acctest.RandInt())
	user2 := fmt.Sprintf("test-user-%d", acctest.RandInt())
	user3 := fmt.Sprintf("test-user-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSPolicyAttachConfig(user1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPolicyAttachmentExists("aws_iam_policy_attachment.test-attach", 3, &out),
					testAccCheckAWSPolicyAttachmentAttributes([]string{user1}, []string{"test-role"}, []string{"test-group"}, &out),
				),
			},
			resource.TestStep{
				Config: testAccAWSPolicyAttachConfigUpdate(user1, user2, user3),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPolicyAttachmentExists("aws_iam_policy_attachment.test-attach", 6, &out),
					testAccCheckAWSPolicyAttachmentAttributes([]string{user3, user3}, []string{"test-role2", "test-role3"}, []string{"test-group2", "test-group3"}, &out),
				),
			},
		},
	})
}

func TestAccAWSPolicyAttachment_paginatedEntities(t *testing.T) {
	var out iam.ListEntitiesForPolicyOutput
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSPolicyAttachmentDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSPolicyPaginatedAttachConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSPolicyAttachmentExists("aws_iam_policy_attachment.test-paginated-attach", 101, &out),
				),
			},
		},
	})
}

func testAccCheckAWSPolicyAttachmentDestroy(s *terraform.State) error {
	return nil
}

func testAccCheckAWSPolicyAttachmentExists(n string, c int64, out *iam.ListEntitiesForPolicyOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No policy name is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).iamconn
		arn := rs.Primary.Attributes["policy_arn"]

		resp, err := conn.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return fmt.Errorf("Error: Policy (%s) not found", n)
		}
		if c != *resp.Policy.AttachmentCount {
			return fmt.Errorf("Error: Policy (%s) has wrong number of entities attached on initial creation", n)
		}
		resp2, err := conn.ListEntitiesForPolicy(&iam.ListEntitiesForPolicyInput{
			PolicyArn: aws.String(arn),
		})
		if err != nil {
			return fmt.Errorf("Error: Failed to get entities for Policy (%s)", arn)
		}

		*out = *resp2
		return nil
	}
}

func testAccCheckAWSPolicyAttachmentAttributes(users []string, roles []string, groups []string, out *iam.ListEntitiesForPolicyOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		uc := len(users)
		rc := len(roles)
		gc := len(groups)

		for _, u := range users {
			for _, pu := range out.PolicyUsers {
				if u == *pu.UserName {
					uc--
				}
			}
		}
		for _, r := range roles {
			for _, pr := range out.PolicyRoles {
				if r == *pr.RoleName {
					rc--
				}
			}
		}
		for _, g := range groups {
			for _, pg := range out.PolicyGroups {
				if g == *pg.GroupName {
					gc--
				}
			}
		}
		if uc != 0 || rc != 0 || gc != 0 {
			return fmt.Errorf("Error: Number of attached users, roles, or groups was incorrect:\n expected %d users and found %d\nexpected %d roles and found %d\nexpected %d groups and found %d", len(users), len(users)-uc, len(roles), len(roles)-rc, len(groups), len(groups)-gc)
		}
		return nil
	}
}

func testAccAWSPolicyAttachConfig(u1 string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "user" {
    name = "%s"
}
resource "aws_iam_role" "role" {
    name = "test-role"
	  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}
resource "aws_iam_group" "group" {
    name = "test-group"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    users = ["${aws_iam_user.user.name}"]
    roles = ["${aws_iam_role.role.name}"]
    groups = ["${aws_iam_group.group.name}"]
    policy_arn = "${aws_iam_policy.policy.arn}"
}`, u1)
}

func testAccAWSPolicyAttachConfigUpdate(u1, u2, u3 string) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "user" {
    name = "%s"
}
resource "aws_iam_user" "user2" {
    name = "%s"
}
resource "aws_iam_user" "user3" {
    name = "%s"
}
resource "aws_iam_role" "role" {
    name = "test-role"
	  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

resource "aws_iam_role" "role2" {
    name = "test-role2"
	  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

}
resource "aws_iam_role" "role3" {
    name = "test-role3"
	  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

}
resource "aws_iam_group" "group" {
    name = "test-group"
}
resource "aws_iam_group" "group2" {
    name = "test-group2"
}
resource "aws_iam_group" "group3" {
    name = "test-group3"
}

resource "aws_iam_policy" "policy" {
    name = "test-policy"
    description = "A test policy"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "iam:ChangePassword"
      ],
      "Resource": "*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_policy_attachment" "test-attach" {
    name = "test-attachment"
    users = [
        "${aws_iam_user.user2.name}",
        "${aws_iam_user.user3.name}"
    ]
    roles = [
        "${aws_iam_role.role2.name}",
        "${aws_iam_role.role3.name}"
    ]
    groups = [
        "${aws_iam_group.group2.name}",
        "${aws_iam_group.group3.name}"
    ]
    policy_arn = "${aws_iam_policy.policy.arn}"
}`, u1, u2, u3)
}

func testAccAWSPolicyPaginatedAttachConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_iam_user" "user" {
	count = 101
	name = "${format("paged-test-user-%d-%%d", count.index + 1)}"
}
resource "aws_iam_policy" "policy" {
	name = "tf-acc-test-policy-%d"
	description = "A test policy"
	policy = <<EOF
{
"Version": "2012-10-17",
"Statement": [
	{
		"Action": [
			"iam:ChangePassword"
		],
		"Resource": "*",
		"Effect": "Allow"
	}
]
}
EOF
}
resource "aws_iam_policy_attachment" "test-paginated-attach" {
	name = "test-attachment"
	users = ["${aws_iam_user.user.*.name}"]
	policy_arn = "${aws_iam_policy.policy.arn}"
}`, rInt, rInt)
}
