# Diamond workflow support.

Diamond workflow allows for a pipeline to scale out the generation of a single bundle. 

A diamond is composed of multiple splits, each split tries to write a set of files, ideally 2 splits do not write the 
same file.

To elaborate, a bundle consists of a filesystem namespace with files in folders, each split can write files to any 
folder but should not try to write the same file.

The individual nodes add files to splits without any co-ordination between them. 
At the time of commit any conflicts due to overlapping files will be dealt with. Conflicts can only occur if the same 
file in the namespace of the bundle is written to by more than one split.

The diamond workflow consists of the following steps.
1. Before a diamond starts, Datamon SDK allows for the creation of an unique ID for a diamond.
   1. This can be a KSUID.
   2. The client is expected to use the KSUID at each step.
   3. command: ```datamon bundle diamond initialize --repo {repo} --message```
      1. Command will output to STDOUT the id for the diamond.
   4. backend model: In the vmetadata bucket the following path will be created.
      ```/diamonds/{repo}/{id}/diamond.yaml```   
1. Each split in a diamond generates a filelist that is stored in the versioned metadata bucket.
   1. Each split is given a client assigned ID. Example split1,split2,split3.. or 1,2,3.. or a set of UUIDs.
   2. On retries the split ID should be the same for a given split.
      1. A node on failure can rerun the same command, as long as the split id is the same
      the previous filelist contents will be superseded by the ones uploaded later.
   3. command: ```datamon bundle diamon add --repo {repo} --id {ID} --split-id {client generated ID for each split in a diamond} --source {path}```
   4. backend model: In the vmetadata bucket the following path will be created or rewritten.
      ```/diamonds/{repo}/{id}/filelist-{split-id}-{ksuid}.yaml```. KSUID suffix generated at start of the 
      add command allows for ordering of uploads from the same split.
   5. This command can also be reused to have incremental uploads before a bundle is created.
   6. The ordering of ```add``` from the same ```--split-id``` can be strongly ordered based on a design similar to WAL.
      This is a nice to have addition and can be added later in the implementation.
1. At the end the file lists are combined into a bundle with the commit message. 
    1. The eventual bundle has a single namespace and if 2 nodes in the diamond have duplicate files, one will win. 
    The other files including the winner will be stored in a special path in the bundle. (Proposal ./.{ID}/conflicts/{orig path}/file-suffix of client ID)
    2. command: ```datamon bundle diamond commit --repo {repo} --id {id} --message {message} --label {label}```
    3. Optionally a commit can be set to fail on conflict.
        1. command: ```datamon bundle diamond commit --repo {repo} --id {id} --message {message} --label {label} --no-confict``` 
    4. A file is written to mark the completion of a diamond and the corresponding bundle ID is written. On failure 2 bundles can be created in the repo.
    for the same diamond but only one will win closing the bundle. ```/diamonds/repo/{id}/bundle.yaml```
## Other commands
 
```datamon bundle diamond list --repo {repo}```

```datamon bundle diamond list-files --repo {repo} --id {id}```

## Alternate proposal

### Skip split-id
The ```split-id``` that is client created can be skipped and replaced by a datamon internally generated KSUID. On retries
a new id would be used. The final merge at the end will deal with ordering the KSUID and picking a winner if there are 
conflicts. 

Pros:

1. Client does not need to create or manage ```split-id```
2. Easy of use (similar to 1)

Cons:
1. Datamon will loose the mapping of client side logic for splits to actual additions to a diamond. ```split-id``` allows
the grouping of which files originated from which split. The client can log the output of the command to capture the details.

### User generated diamond id

Diamond ID is used to 

1. Track splits that are grouped together
2. Separate the namespace of the metadata to allow for concurrent diamonds to the same repo.

Diamond ID can also be client generated which would fail if the same diamond ID is used more than once in the initialize step.

