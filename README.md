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
AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... AWS_COUNTERS_TABLE=bayesian_counters bayes_sampling --port 8080
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
