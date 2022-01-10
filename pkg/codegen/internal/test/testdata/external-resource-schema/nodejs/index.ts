// *** WARNING: this file was generated by test. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

import * as pulumi from "@pulumi/pulumi";
import * as utilities from "./utilities";

// Export members:
export * from "./argFunction";
export * from "./cat";
export * from "./component";
export * from "./foo";
export * from "./provider";
export * from "./workload";

// Export sub-modules:
import * as types from "./types";

export {
    types,
};

// Import resources to register:
import { Cat } from "./cat";
import { Component } from "./component";
import { Foo } from "./foo";
import { Workload } from "./workload";

const _module = {
    version: utilities.getVersion(),
    construct: (name: string, type: string, urn: string): pulumi.Resource => {
        switch (type) {
            case "example::Cat":
                return new Cat(name, <any>undefined, { urn })
            case "example::Component":
                return new Component(name, <any>undefined, { urn })
            case "example::Foo":
                return new Foo(name, <any>undefined, { urn })
            case "example::Workload":
                return new Workload(name, <any>undefined, { urn })
            default:
                throw new Error(`unknown resource type ${type}`);
        }
    },
};
pulumi.runtime.registerResourceModule("example", "", _module)

import { Provider } from "./provider";

pulumi.runtime.registerResourcePackage("example", {
    version: utilities.getVersion(),
    constructProvider: (name: string, type: string, urn: string): pulumi.ProviderResource => {
        if (type !== "pulumi:providers:example") {
            throw new Error(`unknown provider type ${type}`);
        }
        return new Provider(name, <any>undefined, { urn });
    },
});
