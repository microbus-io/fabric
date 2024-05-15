# Package `env`

Package `env` manages the loading of environment variables.
Variables are first searched for in an in-memory stack, then in a file `env.yaml` in the current working directory, and finally in the OS.
The in-memory stack provides `Push` and `Pop` operations and is intended for modifying the environment during unit and integration testing.
The YAML file allows the setting of environment variables in a file that can be shared and version-controlled.
