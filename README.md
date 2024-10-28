# Bayes Sampling

A trivial implementation of Thompson sampling algorithm for multi-armed bandit problem with binary outcomes.

To better understand what happens here, I recommend the following
course: https://www.udemy.com/course/bayesian-machine-learning-in-python-ab-testing/

Created by Alexey Shagraev, 22 October 2024

## Setup

This program uses [AWS DynamoDB](https://docs.aws.amazon.com/dynamodb/) tables to store the counters.

**1. Create the table**

```
aws dynamodb create-table \
--table-name bayesian_counters \
--billing-mode PAY_PER_REQUEST \
--key-schema AttributeName=k,KeyType=HASH \
--attribute-definitions AttributeName=k,AttributeType=S
```

Here, we only need a single partition key with the name `key` of type string. I recommend
using [on-demand](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/on-demand-capacity-mode.html
) tables at the start since the required capacity is quite hard to predict before the system is up and running
processing production traffic.

**2. Run the program**

The program requires AWS credentials to be available in the ENV using the same names as AWS CLI. It also requires the
ENV variable `AWS_COUNTERS_TABLE` to be set to the name of the table that we utilize for storing the counters. So in the
simplest variant we can launch it in the following way:

```
AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_REGION=... AWS_COUNTERS_TABLE=bayesian_counters bayes_sampling --port 8080
```

Of course, you're free to set other port if necessary.

**3. Populate data: online mode**

The first way to update the counter values is online mode. Say, you have a website which populates the user events
continuously, then you can populate the `view` event when the user reaches a certain page, and the `click` event when
the conversion happened (e.g., when the user performed a certain action on the page).

Let's say we have a key named `test_key_1` and a couple of views happened as well as a single click:

```
curl "localhost:8080/add_view?key=test_key_1"
{
  "Key": "test_key_1_views",
  "Value": 1
}
curl "localhost:8080/add_view?key=test_key_1"
{
  "Key": "test_key_1_views",
  "Value": 2
}
curl "localhost:8080/add_click?key=test_key_1"
{
  "Key": "test_key_1_clicks",
  "Value": 1
}
```

This mode might be useful in systems where nearly real-time updates for the counters are needed: when you'd like to
update the data as soon as it arrives and then sample using the updated values right away. This might be the case when
many of the variants are expected to perform poorly, and you want to minimize the time of them being exposed to the
users.

**4. Populate data: batch mode**

Instead of online mode, you can use batch-style way of populating data — say, running some SQL queries over your
analytical storage from time to time. This is also a preferable way for cases when the raw signals require some kind of
processing before being put into the system — such as cleaning against robots or fraud events.

At the same time, naturally, this way of populating the data introduces some delays in the counter updates.

In the batch mode, one could populate the clicks and views data with a single request each:

```
curl "localhost:8080/set_views?key=test_key_2&views=3"
{
  "Key": "test_key_2_views",
  "Value": 3
}

curl "localhost:8080/set_clicks?key=test_key_2&clicks=2"
{
  "Key": "test_key_2_clicks",
  "Value": 2
}
```

**5. Perform sampling**

After populating the views and clicks data for the keys, we can perform sampling to decide which key to use when the
next user shows up. Let's run the sampling a few times:

```
curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_2",
  "sampled_score": 0.8721687832630763,
  "sampled_values": {
    "test_key_1": 0.2076339591012726,
    "test_key_2": 0.8721687832630763
  }
}

curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_2",
  "sampled_score": 0.5414033336405559,
  "sampled_values": {
    "test_key_1": 0.4332324438396514,
    "test_key_2": 0.5414033336405559
  }
}

curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_1",
  "sampled_score": 0.3446424101049709,
  "sampled_values": {
    "test_key_1": 0.3446424101049709,
    "test_key_2": 0.2703417652845729
  }
}

curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_1",
  "sampled_score": 0.543663689858079,
  "sampled_values": {
    "test_key_1": 0.543663689858079,
    "test_key_2": 0.3576722223707369
  }
}

curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_2",
  "sampled_score": 0.3674973672551862,
  "sampled_values": {
    "test_key_1": 0.2698637410004653,
    "test_key_2": 0.3674973672551862
  }
}

curl "localhost:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_2",
  "sampled_score": 0.6606418928406255,
  "sampled_values": {
    "test_key_1": 0.22560511773124498,
    "test_key_2": 0.6606418928406255
  }
}
```

As you can see, in this example the key `test_key_2` has been sampled more frequently than `test_key_1`, and that's to
be expected since its current estimation of conversion rate is 2 out of 3 against 1 out of 2. At the same time, for now,
the level of certainty in the conversion rates is relatively low, so `test_key_1` will be sampled from time to time.

To select one key out of the list of the keys passed to the `/sample` endpoint, the following algorithm is performed:

- For each key, we have the `clicks` and `views` counters
- If `clicks > views`, then set `clicks = views` to avoid uncertainty
- Sample from the [Beta distribution](https://en.wikipedia.org/wiki/Beta_distribution) using parameters:
    - `alpha = clicks + 1`
    - `beta = views - clicks + 1`
- The key with the highest sampled value wins.

## Running in Docker on AWS

**1. Release**

```
docker build --platform linux/amd64 \
--build-arg TARGETOS=linux \
--build-arg TARGETARCH=amd64 \
-t ashagraev/bayes_optimization:linux-amd64-latest .
docker push ashagraev/bayes_optimization:linux-amd64-latest

docker build --platform linux/amd64 \
--build-arg TARGETOS=linux \
--build-arg TARGETARCH=amd64 \
-t ashagraev/bayes_optimization:linux-amd64-v001 .
docker push ashagraev/bayes_optimization:linux-amd64-v001
```

**2. Launching on a VM**

```
docker pull ashagraev/bayes_optimization:linux-amd64-v001

docker run -d -p 8080:8080 \
-e AWS_ACCESS_KEY_ID=... \
-e AWS_SECRET_ACCESS_KEY=... \
-e AWS_REGION=... \
-e AWS_COUNTERS_TABLE=bayesian_counters \
ashagraev/bayes_optimization:linux-amd64-v001
```

**3. Launching in Fargate**

Assuming, you don't have a pre-existing Fargate cluster, we start with creating one:

```
aws ecs create-cluster --cluster-name fargate-test
```

Then we create a Fargate task definition that would utilize our Docker Hub repository. Please
substitute `YOUR_AWS_ACCOUNT_ID` with your AWS account ID in the following command:

```
aws ecs register-task-definition \
  --family bayes-optimization \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu "512" \
  --memory "1024" \
  --execution-role-arn arn:aws:iam::YOUR_AWS_ACCOUNT_ID:role/ecsTaskExecutionRole \
  --container-definitions '[
    {
      "name": "bayes-optimization",
      "image": "ashagraev/bayes_optimization:linux-amd64-v001",
      "cpu": 512,
      "memory": 1024,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "AWS_ACCESS_KEY_ID",
          "value": "..."
        },
        {
          "name": "AWS_SECRET_ACCESS_KEY",
          "value": "..."
        },
        {
          "name": "AWS_REGION",
          "value": "..."
        },
        {
          "name": "AWS_COUNTERS_TABLE",
          "value": "bayesian_counters"
        }
      ]
    }
  ]'
```

Also, please figure a better way to pass the ENV variables with the AWS credentials inside the container, which will be
better aligned with your infrastructure.

Here, we're using minimalistic configuration with 0.5 vCPU and 1Gb RAM as the service does not require a ton of
resources. Lastly, we create the Fargate service. The easiest way to go about it is to first retrieve the default
subnets list:

```
aws ec2 describe-subnets --filters Name=default-for-az,Values=true --query 'Subnets[*].SubnetId' --output text
```

Then use this list in the following `aws ecs create-service` command:

```
aws ecs create-service \
--cluster fargate-test \
--service-name bayes-optimization \
--task-definition bayes-optimization \
--launch-type FARGATE \
--desired-count 1 \
--network-configuration '{
  "awsvpcConfiguration": {
    "subnets": ["subnet-xxxxxxxx", "subnet-yyyyyyyy", "subnet-zzzzzzzz"],
    "assignPublicIp": "ENABLED"
  }
}' \
--platform-version "LATEST"
```

This will launch a simple Fargate service with a single instance. You can now attach a load balancer to it and play with
the configuration as you wish: split the capacity between `FARGATE` and `FARGATE_SPOT`, change the number of instances,
and so on.

After the service is set up, you can send requests to it. Assuming there's no balancer yet, but the tasks have public
addresses,
one can simply query a single task to see if the service is working:

```
curl "http://aaa.bbb.ccc.ddd:8080/health"
OK

curl "http://aaa.bbb.ccc.ddd:8080/sample?key=test_key_1&key=test_key_2"
{
  "sampled_key": "test_key_1",
  "sampled_score": 0.7529484868782941,
  "sampled_values": {
    "test_key_1": 0.7529484868782941,
    "test_key_2": 0.687636170763479
  }
}
```

## Visualizing Distributions

The program also allows for extracting the Beta distribution parameters for all the interesting keys. This way, the user
can visualize them to get an intel on what are the conversion rates and confidences collected so far. Here's an example
of a Python Notebook code that would collect the distribution parameters from newly created Fargate service and
visualize them:

```python
import matplotlib.pyplot as plt
from matplotlib.pyplot import figure
import numpy as np
from scipy.stats import beta
import requests

keys = [
    'test_key_1',
    'test_key_2',
]
endpoint = 'http://aaa.bbb.ccc.ddd:8080/distribution_params?key=' + '&key='.join(keys)
beta_params = requests.get(endpoint).json()

figure(figsize=(20, 10), dpi=80)

x = np.linspace(0,1,1000)

cmap = plt.get_cmap('gnuplot')
colors = [cmap(i) for i in np.linspace(0.5, 1.0, len(keys))]

plt.rcParams['axes.facecolor'] = '#212121'
plt.rcParams['figure.facecolor'] = '#212121'
plt.rcParams['lines.linewidth'] = '2'
plt.rcParams['axes.prop_cycle'] = plt.cycler('color',['#AAAAAA'])
plt.rcParams['text.color'] = '#FFFFFF'
plt.rcParams['grid.color'] = '#AAAAAA'
plt.rcParams['xtick.color'] = '#AAAAAA'
plt.rcParams['ytick.color'] = '#AAAAAA'
plt.rcParams['grid.color'] = '#AAAAAA'
plt.rcParams['xtick.color'] = '#AAAAAA'
plt.rcParams['axes.spines.left'] = False
plt.rcParams['axes.spines.top'] = False
plt.rcParams['axes.spines.right'] = False
plt.rcParams['axes.edgecolor'] = '#AAAAAA'

for idx, key in enumerate(keys):
    plt.plot(x, beta.pdf(x, beta_params[idx]['alpha'], beta_params[idx]['beta']), color=colors[idx], label=key)

plt.yticks([])
plt.legend(loc="upper left")
plt.show()
```

As you can see, with the current numbers, the uncertainty is quite high. If we change the number of events, though, the
situation will become different:

```
test_key_1: clicks = 10, views = 20
test_key_2: clicks = 20, views = 30

curl "http://aaa.bbb.ccc.ddd:8080/set_views?key=test_key_2&views=30"
curl "http://aaa.bbb.ccc.ddd:8080/set_clicks?key=test_key_2&clicks=20"
curl "http://aaa.bbb.ccc.ddd:8080/set_views?key=test_key_1&views=20"
curl "http://aaa.bbb.ccc.ddd:8080/set_clicks?key=test_key_1&clicks=10"
```

```
test_key_1: clicks = 100, views = 200
test_key_2: clicks = 200, views = 300

curl "http://aaa.bbb.ccc.ddd:8080/set_views?key=test_key_2&views=300"
curl "http://aaa.bbb.ccc.ddd:8080/set_clicks?key=test_key_2&clicks=200"
curl "http://aaa.bbb.ccc.ddd:8080/set_views?key=test_key_1&views=200"
curl "http://aaa.bbb.ccc.ddd:8080/set_clicks?key=test_key_1&clicks=100"
```

As you can see, as the number of views growth, the confidence also growth as the distributions become thinner. In
real-time applications, the more conversions a certain variant displays, the more often it is presented to the users, so
you can expect the distributions to become thinner from left to right: higher conversion rate leads to more confidence.
