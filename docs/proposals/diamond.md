# Diamond workflow support.

Diamond workflow allows for a pipeline to scale out the generation of a single bundle. The individual nodes writes to 
splits of the eventual bundle without any co-ordination between them. At the time of commit any conflicts will be
dealt with. Conflicts can only occur if the same file in the namespace of the bundle is written to by more than one split.

The diamond workflow consists of the following steps.
1. Before a diamond starts, Datamon SDK allows for the creation of an unique ID for a diamond.
   1. This can be a KSUID.
   2. The client is expected to use the KSUID at each step.
   3. command: ```datamon bundle split initialize --repo <repo> --message```
      1. Command will output to STDOUT the id for the diamond.
   4. backend model: In the vmetadata bucket the following path will be created.
      ```/splits/repo/<id>/split.yaml```   
1. Each split in a diamond generates a filelist that is stored in the versioned metadata folder.
   1. Each split is given a client assigned ID. Example split1,split2,split3.. or 1,2,3.. or a set of UUIDs.
   2. On retries the split ID should be the same for a given split.
      1. A node on failure can rerun the same command, as long as the split id is the same
      the previous filelist contents will be superseded by the ones uploaded later.
   3. command: ```datamon bundle split add --repo <repo> --split-id <client generated id> --source <path>```
   4. backend model: In the vmetadata bucket the following path will be created or rewritten.
      ```/splits/repo/<id>/filelist-<split-id>-<ksuid>.yaml```. KSUID suffix generated at start of the 
      add command allows for ordering of uploads from the same split.
   5. This command can also be reused to have incremental uploads before a bundle is created.
   6. The ordering of ```add``` from the same ```--split-id``` can be strongly ordered based on a design similar to WAL.
      This is a nice to have addition and can be added later in the implementation.
1. At the end the file lists are combined into a bundle with the commit message. 
    1. The eventual bundle has a single namespace and if 2 nodes in the diamond have duplicate files, one will win. 
    The other files including the winner will be stored in a special path in the bundle. (Proposal ./.<ID>/conflicts/<orig path>/file-suffix of client ID)
    2. command: ```datamon bundle split commit --repo <repo> --split-id <id> --message <message> --label <label>```
    3. Optionally a commit can be set to fail on conflict.
        1. command: ```datamon bundle split commit --repo <repo> --split-id <id> --message <message> --label <label> --no-confict``` 

## Other commands
 
```datamon bundle split list --repo <repo>```

```datamon bundle split list-files --repo <repo> --split-id <id>```

