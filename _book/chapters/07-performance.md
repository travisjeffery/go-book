# Write fast Go

## Why do I want to read this?

More than ever it's important to build fast applications. Since we're often building backend service acting as the core of our larger projects if they're slow so is our whole project.

Every nanosecond counts and is often the difference beteween a user using your service or not. Amazon has research showing that if their web pages took [1 second longer to load it would cost them 1.6 billion each year](https://www.fastcompany.com/1825005/how-one-second-could-cost-amazon-16-billion-sales), each []100ms cost them 1% in sales](https://blog.gigaspaces.com/amazon-found-every-100ms-of-latency-cost-them-1-in-sales/). [Pinterest found that a 40% reduction in perceived wait time increased signups by 15%](https://medium.com/@Pinterest_Engineering/driving-user-growth-with-performance-improvements-cfc50dafadd7). [The BBC found they lost 10% of users for every additional second of load time](https://www.creativebloq.com/features/how-the-bbc-builds-websites-that-scale). Google found over 50% of mobile users abandoned web pages if they took 3 seconds or more.

## What will I learn?

- How to make performant Go code that uses memory efficiently.
- The difference between memory on the stack and heap when you want or need either, how to write Go code to control where you're allocating memory.
- How to write Go code that doesn't need to garbage collect unless necessary.

## What will I be able to do that I couldn’t do before?

## Where are we going next, and how does this fit in?
