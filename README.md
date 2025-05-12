# intro
Here will be some observations.

## motivation
* learn
* get better


### where we can see packets

so there are some tools that Brendan Gregg has mentioned [here](https://github.com/brendangregg/perf-tools).
lets see what can be useful to take a closer look at data at various points.

We have two options how to initiate request we will inspect - via browser or curl, most simple one in terms of traceability would be curl with which we will proceed with an example. most obvious choice imho for checking what is happening under the hood is strace. as we are mac then from what I read dtrace is a go to tool on bsd systems - here is a good talk on this tool by charismatic bryan cantrill- https://www.youtube.com/watch?v=TgmA48fILq8. 