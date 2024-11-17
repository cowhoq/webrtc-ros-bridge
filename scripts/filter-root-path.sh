#!/bin/bash

# List of packages to include
packages=(
    "rcl"
    "sensor_msgs"
    "builtin_interfaces"
    "geometry_msgs"
    "std_msgs"
    "rosidl_runtime_c"
    "rcl_action"
    "action_msgs"
    "unique_identifier_msgs"
    "rosidl_typesupport_interface"
    "rcutils"
)

# Generate the grep pattern
include_pattern=$(printf "install/%s$|" "${packages[@]}")
include_pattern=${exclude_pattern%|} # Remove the trailing '|'

# Step 1: Echo the AMENT_PREFIX_PATH
echo "$AMENT_PREFIX_PATH" |
# Step 2: Replace all ":" with new lines
tr ':' '\n' |
# Step 3: Select needed packages
grep -E "$include_pattern" |
# Step 4: Replace all new lines back with ":"
tr '\n' ':' |
# Remove trailing ":" if present
sed 's/:$//'
