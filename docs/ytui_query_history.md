## ytui query history

Search for videos from your history

### Synopsis


Search for videos from your history. Due to Youtube Data APIv3 not allowing to retrieve user history,
ytui will feed and store its own history in a json file in the configuration directory. Any video watched with ytui
will be stored in there.

```
ytui query history [flags]
```

### Options

```
  -h, --help   help for history
```

### Options inherited from parent commands

```
  -d, --download           Download the selected video instead of watching it
  -l, --log-level string   Override log level (debug, info, error)
```

### SEE ALSO

* [ytui query](ytui_query.md)	 - Run queries for videos through different patterns

###### Auto generated by spf13/cobra on 30-Sep-2024
