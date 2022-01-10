# coding=utf-8
# *** WARNING: this file was generated by test. ***
# *** Do not edit by hand unless you're certain you know what you are doing! ***

from . import _utilities
import typing
# Export this package's modules as members:
from .arg_function import *
from .cat import *
from .component import *
from .foo import *
from .provider import *
from .workload import *
from ._inputs import *
_utilities.register(
    resource_modules="""
[
 {
  "pkg": "example",
  "mod": "",
  "fqn": "pulumi_example",
  "classes": {
   "example::Cat": "Cat",
   "example::Component": "Component",
   "example::Foo": "Foo",
   "example::Workload": "Workload"
  }
 }
]
""",
    resource_packages="""
[
 {
  "pkg": "example",
  "token": "pulumi:providers:example",
  "fqn": "pulumi_example",
  "class": "Provider"
 }
]
"""
)
