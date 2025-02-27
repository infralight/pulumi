// *** WARNING: this file was generated by test. ***
// *** Do not edit by hand unless you're certain you know what you are doing! ***

using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Threading.Tasks;
using Pulumi.Serialization;

namespace Pulumi.Example
{
    [ExampleResourceType("example::Foo")]
    public partial class Foo : Pulumi.ComponentResource
    {
        /// <summary>
        /// Create a Foo resource with the given unique name, arguments, and options.
        /// </summary>
        ///
        /// <param name="name">The unique name of the resource</param>
        /// <param name="args">The arguments used to populate this resource's properties</param>
        /// <param name="options">A bag of options that control this resource's behavior</param>
        public Foo(string name, FooArgs? args = null, ComponentResourceOptions? options = null)
            : base("example::Foo", name, args ?? new FooArgs(), MakeResourceOptions(options, ""), remote: true)
        {
        }

        private static ComponentResourceOptions MakeResourceOptions(ComponentResourceOptions? options, Input<string>? id)
        {
            var defaultOptions = new ComponentResourceOptions
            {
                Version = Utilities.Version,
            };
            var merged = ComponentResourceOptions.Merge(defaultOptions, options);
            // Override the ID if one was specified for consistency with other language SDKs.
            merged.Id = id ?? merged.Id;
            return merged;
        }

        public Pulumi.Output<string> GetKubeconfig(FooGetKubeconfigArgs? args = null)
            => Pulumi.Deployment.Instance.Call<FooGetKubeconfigResult>("example::Foo/getKubeconfig", args ?? new FooGetKubeconfigArgs(), this).Apply(v => v.Kubeconfig);
    }

    public sealed class FooArgs : Pulumi.ResourceArgs
    {
        public FooArgs()
        {
        }
    }

    /// <summary>
    /// The set of arguments for the <see cref="Foo.GetKubeconfig"/> method.
    /// </summary>
    public sealed class FooGetKubeconfigArgs : Pulumi.CallArgs
    {
        [Input("profileName")]
        public Input<string>? ProfileName { get; set; }

        [Input("roleArn")]
        public Input<string>? RoleArn { get; set; }

        public FooGetKubeconfigArgs()
        {
        }
    }

    /// <summary>
    /// The results of the <see cref="Foo.GetKubeconfig"/> method.
    /// </summary>
    [OutputType]
    internal sealed class FooGetKubeconfigResult
    {
        public readonly string Kubeconfig;

        [OutputConstructor]
        private FooGetKubeconfigResult(string kubeconfig)
        {
            Kubeconfig = kubeconfig;
        }
    }
}
