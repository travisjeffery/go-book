# Make servers distributed, reliable, and scalable

## Why do I want to read this?

Services must be distributed to be reliable and handle faulty hardware or network partitions (which often outside of your control).

If your service isn't fault tolerant and reliable your service will eventually go down and your users will stop using it. People don't want to waste their time or money on something they can't rely on. Regardless of what type of project or company you work at this means bad news. It could mean the end of your startup or the end of your project at a larger company.

As use of your service grows you want to be able to seamlessly scale it up. This means we can handle
the increasing usage, and also means we can match resource allocation to usage from the start and
not over provision and over spend.

## What will I learn?

- How to build a distributed service from the ground up with built-in service discovery and consensus, i.e. how to make instances of your service running on separate computers find each other and achieve consensus on what they're doing.
- What service discovery and consensus are, where they're used, and what for.

## What will I be able to do that I couldn’t do before?

- Build your own distributed services including built-in service discovery and consensus that are fault tolerant and scalable.
- Debug your distributed services you've built or use.

## Where are we going next, and how does this fit in?

We've have written our service and have it running on our computer. Now we're going to deploy our service to web using Kubernete so people can use our service from anywhere.
