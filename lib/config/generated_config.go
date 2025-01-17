// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

var DefaultYAML = []byte(`# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Do not use this file for site configuration. Create
# /etc/arvados/config.yml instead.
#
# The order of precedence (highest to lowest):
# 1. Legacy component-specific config files (deprecated)
# 2. /etc/arvados/config.yml
# 3. config.default.yml

Clusters:
  xxxxx:
    SystemRootToken: ""

    # Token to be included in all healthcheck requests. Disabled by default.
    # Server expects request header of the format "Authorization: Bearer xxx"
    ManagementToken: ""

    Services:

      # In each of the service sections below, the keys under
      # InternalURLs are the endpoints where the service should be
      # listening, and reachable from other hosts in the cluster.
      SAMPLE:
        InternalURLs:
          "http://example.host:12345": {}
          SAMPLE: {}
        ExternalURL: "-"

      RailsAPI:
        InternalURLs: {}
        ExternalURL: "-"
      Controller:
        InternalURLs: {}
        ExternalURL: ""
      Websocket:
        InternalURLs: {}
        ExternalURL: ""
      Keepbalance:
        InternalURLs: {}
        ExternalURL: "-"
      GitHTTP:
        InternalURLs: {}
        ExternalURL: ""
      GitSSH:
        InternalURLs: {}
        ExternalURL: ""
      DispatchCloud:
        InternalURLs: {}
        ExternalURL: "-"
      SSO:
        InternalURLs: {}
        ExternalURL: ""
      Keepproxy:
        InternalURLs: {}
        ExternalURL: ""
      WebDAV:
        InternalURLs: {}
        # Base URL for Workbench inline preview.  If blank, use
        # WebDAVDownload instead, and disable inline preview.
        # If both are empty, downloading collections from workbench
        # will be impossible.
        #
        # It is important to properly configure the download service
        # to migitate cross-site-scripting (XSS) attacks.  A HTML page
        # can be stored in collection.  If an attacker causes a victim
        # to visit that page through Workbench, it will be rendered by
        # the browser.  If all collections are served at the same
        # domain, the browser will consider collections as coming from
        # the same origin and having access to the same browsing data,
        # enabling malicious Javascript on that page to access Arvados
        # on behalf of the victim.
        #
        # This is mitigating by having separate domains for each
        # collection, or limiting preview to circumstances where the
        # collection is not accessed with the user's regular
        # full-access token.
        #
        # Serve preview links using uuid or pdh in subdomain
        # (requires wildcard DNS and TLS certificate)
        #   https://*.collections.uuid_prefix.arvadosapi.com
        #
        # Serve preview links using uuid or pdh in main domain
        # (requires wildcard DNS and TLS certificate)
        #   https://*--collections.uuid_prefix.arvadosapi.com
        #
        # Serve preview links by setting uuid or pdh in the path.
        # This configuration only allows previews of public data or
        # collection-sharing links, because these use the anonymous
        # user token or the token is already embedded in the URL.
        # Other data must be handled as downloads via WebDAVDownload:
        #   https://collections.uuid_prefix.arvadosapi.com
        #
        ExternalURL: ""

      WebDAVDownload:
        InternalURLs: {}
        # Base URL for download links. If blank, serve links to WebDAV
        # with disposition=attachment query param.  Unlike preview links,
        # browsers do not render attachments, so there is no risk of XSS.
        #
        # If WebDAVDownload is blank, and WebDAV uses a
        # single-origin form, then Workbench will show an error page
        #
        # Serve download links by setting uuid or pdh in the path:
        #   https://download.uuid_prefix.arvadosapi.com
        #
        ExternalURL: ""

      Keepstore:
        InternalURLs: {}
        ExternalURL: "-"
      Composer:
        InternalURLs: {}
        ExternalURL: ""
      WebShell:
        InternalURLs: {}
        # ShellInABox service endpoint URL for a given VM.  If empty, do not
        # offer web shell logins.
        #
        # E.g., using a path-based proxy server to forward connections to shell hosts:
        # https://webshell.uuid_prefix.arvadosapi.com
        #
        # E.g., using a name-based proxy server to forward connections to shell hosts:
        # https://*.webshell.uuid_prefix.arvadosapi.com
        ExternalURL: ""
      Workbench1:
        InternalURLs: {}
        ExternalURL: ""
      Workbench2:
        InternalURLs: {}
        ExternalURL: ""
      Nodemanager:
        InternalURLs: {}
        ExternalURL: "-"
      Health:
        InternalURLs: {}
        ExternalURL: "-"

    PostgreSQL:
      # max concurrent connections per arvados server daemon
      ConnectionPool: 32
      Connection:
        # All parameters here are passed to the PG client library in a connection string;
        # see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-PARAMKEYWORDS
        host: ""
        port: ""
        user: ""
        password: ""
        dbname: ""
        SAMPLE: ""
    API:
      # Maximum size (in bytes) allowed for a single API request.  This
      # limit is published in the discovery document for use by clients.
      # Note: You must separately configure the upstream web server or
      # proxy to actually enforce the desired maximum request size on the
      # server side.
      MaxRequestSize: 134217728

      # Limit the number of bytes read from the database during an index
      # request (by retrieving and returning fewer rows than would
      # normally be returned in a single response).
      # Note 1: This setting never reduces the number of returned rows to
      # zero, no matter how big the first data row is.
      # Note 2: Currently, this is only checked against a specific set of
      # columns that tend to get large (collections.manifest_text,
      # containers.mounts, workflows.definition). Other fields (e.g.,
      # "properties" hashes) are not counted against this limit.
      MaxIndexDatabaseRead: 134217728

      # Maximum number of items to return when responding to a APIs that
      # can return partial result sets using limit and offset parameters
      # (e.g., *.index, groups.contents). If a request specifies a "limit"
      # parameter higher than this value, this value is used instead.
      MaxItemsPerResponse: 1000

      # API methods to disable. Disabled methods are not listed in the
      # discovery document, and respond 404 to all requests.
      # Example: {"jobs.create":{}, "pipeline_instances.create": {}}
      DisabledAPIs: {}

      # Interval (seconds) between asynchronous permission view updates. Any
      # permission-updating API called with the 'async' parameter schedules a an
      # update on the permission view in the future, if not already scheduled.
      AsyncPermissionsUpdateInterval: 20s

      # Maximum number of concurrent outgoing requests to make while
      # serving a single incoming multi-cluster (federated) request.
      MaxRequestAmplification: 4

      # RailsSessionSecretToken is a string of alphanumeric characters
      # used by Rails to sign session tokens. IMPORTANT: This is a
      # site secret. It should be at least 50 characters.
      RailsSessionSecretToken: ""

      # Maximum wall clock time to spend handling an incoming request.
      RequestTimeout: 5m

      # Websocket will send a periodic empty event after 'SendTimeout'
      # if there is no other activity to maintain the connection /
      # detect dropped connections.
      SendTimeout: 60s

      WebsocketClientEventQueue: 64
      WebsocketServerEventQueue: 4

      # Timeout on requests to internal Keep services.
      KeepServiceRequestTimeout: 15s

    Users:
      # Config parameters to automatically setup new users.  If enabled,
      # this users will be able to self-activate.  Enable this if you want
      # to run an open instance where anyone can create an account and use
      # the system without requiring manual approval.
      #
      # The params AutoSetupNewUsersWith* are meaningful only when AutoSetupNewUsers is turned on.
      # AutoSetupUsernameBlacklist is a list of usernames to be blacklisted for auto setup.
      AutoSetupNewUsers: false
      AutoSetupNewUsersWithVmUUID: ""
      AutoSetupNewUsersWithRepository: false
      AutoSetupUsernameBlacklist:
        arvados: {}
        git: {}
        gitolite: {}
        gitolite-admin: {}
        root: {}
        syslog: {}
        SAMPLE: {}

      # When NewUsersAreActive is set to true, new users will be active
      # immediately.  This skips the "self-activate" step which enforces
      # user agreements.  Should only be enabled for development.
      NewUsersAreActive: false

      # The e-mail address of the user you would like to become marked as an admin
      # user on their first login.
      # In the default configuration, authentication happens through the Arvados SSO
      # server, which uses OAuth2 against Google's servers, so in that case this
      # should be an address associated with a Google account.
      AutoAdminUserWithEmail: ""

      # If AutoAdminFirstUser is set to true, the first user to log in when no
      # other admin users exist will automatically become an admin user.
      AutoAdminFirstUser: false

      # Email address to notify whenever a user creates a profile for the
      # first time
      UserProfileNotificationAddress: ""
      AdminNotifierEmailFrom: arvados@example.com
      EmailSubjectPrefix: "[ARVADOS] "
      UserNotifierEmailFrom: arvados@example.com
      NewUserNotificationRecipients: {}
      NewInactiveUserNotificationRecipients: {}

      # Set AnonymousUserToken to enable anonymous user access. You can get
      # the token by running "bundle exec ./script/get_anonymous_user_token.rb"
      # in the directory where your API server is running.
      AnonymousUserToken: ""

    AuditLogs:
      # Time to keep audit logs, in seconds. (An audit log is a row added
      # to the "logs" table in the PostgreSQL database each time an
      # Arvados object is created, modified, or deleted.)
      #
      # Currently, websocket event notifications rely on audit logs, so
      # this should not be set lower than 300 (5 minutes).
      MaxAge: 336h

      # Maximum number of log rows to delete in a single SQL transaction.
      #
      # If MaxDeleteBatch is 0, log entries will never be
      # deleted by Arvados. Cleanup can be done by an external process
      # without affecting any Arvados system processes, as long as very
      # recent (<5 minutes old) logs are not deleted.
      #
      # 100000 is a reasonable batch size for most sites.
      MaxDeleteBatch: 0

      # Attributes to suppress in events and audit logs.  Notably,
      # specifying {"manifest_text": {}} here typically makes the database
      # smaller and faster.
      #
      # Warning: Using any non-empty value here can have undesirable side
      # effects for any client or component that relies on event logs.
      # Use at your own risk.
      UnloggedAttributes: {}

    SystemLogs:

      # Logging threshold: panic, fatal, error, warn, info, debug, or
      # trace
      LogLevel: info

      # Logging format: json or text
      Format: json

      # Maximum characters of (JSON-encoded) query parameters to include
      # in each request log entry. When params exceed this size, they will
      # be JSON-encoded, truncated to this size, and logged as
      # params_truncated.
      MaxRequestLogParamsSize: 2000

    Collections:
      # Allow clients to create collections by providing a manifest with
      # unsigned data blob locators. IMPORTANT: This effectively disables
      # access controls for data stored in Keep: a client who knows a hash
      # can write a manifest that references the hash, pass it to
      # collections.create (which will create a permission link), use
      # collections.get to obtain a signature for that data locator, and
      # use that signed locator to retrieve the data from Keep. Therefore,
      # do not turn this on if your users expect to keep data private from
      # one another!
      BlobSigning: true

      # BlobSigningKey is a string of alphanumeric characters used to
      # generate permission signatures for Keep locators. It must be
      # identical to the permission key given to Keep. IMPORTANT: This is
      # a site secret. It should be at least 50 characters.
      #
      # Modifying BlobSigningKey will invalidate all existing
      # signatures, which can cause programs to fail (e.g., arv-put,
      # arv-get, and Crunch jobs).  To avoid errors, rotate keys only when
      # no such processes are running.
      BlobSigningKey: ""

      # Default replication level for collections. This is used when a
      # collection's replication_desired attribute is nil.
      DefaultReplication: 2

      # Lifetime (in seconds) of blob permission signatures generated by
      # the API server. This determines how long a client can take (after
      # retrieving a collection record) to retrieve the collection data
      # from Keep. If the client needs more time than that (assuming the
      # collection still has the same content and the relevant user/token
      # still has permission) the client can retrieve the collection again
      # to get fresh signatures.
      #
      # This must be exactly equal to the -blob-signature-ttl flag used by
      # keepstore servers.  Otherwise, reading data blocks and saving
      # collections will fail with HTTP 403 permission errors.
      #
      # Modifying BlobSigningTTL invalidates existing signatures; see
      # BlobSigningKey note above.
      #
      # The default is 2 weeks.
      BlobSigningTTL: 336h

      # Default lifetime for ephemeral collections: 2 weeks. This must not
      # be less than BlobSigningTTL.
      DefaultTrashLifetime: 336h

      # Interval (seconds) between trash sweeps. During a trash sweep,
      # collections are marked as trash if their trash_at time has
      # arrived, and deleted if their delete_at time has arrived.
      TrashSweepInterval: 60s

      # If true, enable collection versioning.
      # When a collection's preserve_version field is true or the current version
      # is older than the amount of seconds defined on PreserveVersionIfIdle,
      # a snapshot of the collection's previous state is created and linked to
      # the current collection.
      CollectionVersioning: false

      #   0s = auto-create a new version on every update.
      #  -1s = never auto-create new versions.
      # > 0s = auto-create a new version when older than the specified number of seconds.
      PreserveVersionIfIdle: -1s

      # Managed collection properties. At creation time, if the client didn't
      # provide the listed keys, they will be automatically populated following
      # one of the following behaviors:
      #
      # * UUID of the user who owns the containing project.
      #   responsible_person_uuid: {Function: original_owner, Protected: true}
      #
      # * Default concrete value.
      #   foo_bar: {Value: baz, Protected: false}
      #
      # If Protected is true, only an admin user can modify its value.
      ManagedProperties:
        SAMPLE: {Function: original_owner, Protected: true}

      # In "trust all content" mode, Workbench will redirect download
      # requests to WebDAV preview link, even in the cases when
      # WebDAV would have to expose XSS vulnerabilities in order to
      # handle the redirect (see discussion on Services.WebDAV).
      #
      # This setting has no effect in the recommended configuration,
      # where the WebDAV is configured to have a separate domain for
      # every collection; in this case XSS protection is provided by
      # browsers' same-origin policy.
      #
      # The default setting (false) is appropriate for a multi-user site.
      TrustAllContent: false

      # Cache parameters for WebDAV content serving:
      # * TTL: Maximum time to cache manifests and permission checks.
      # * UUIDTTL: Maximum time to cache collection state.
      # * MaxBlockEntries: Maximum number of block cache entries.
      # * MaxCollectionEntries: Maximum number of collection cache entries.
      # * MaxCollectionBytes: Approximate memory limit for collection cache.
      # * MaxPermissionEntries: Maximum number of permission cache entries.
      # * MaxUUIDEntries: Maximum number of UUID cache entries.
      WebDAVCache:
        TTL: 300s
        UUIDTTL: 5s
        MaxBlockEntries:      4
        MaxCollectionEntries: 1000
        MaxCollectionBytes:   100000000
        MaxPermissionEntries: 1000
        MaxUUIDEntries:       1000

    Login:
      # These settings are provided by your OAuth2 provider (eg
      # Google) used to perform upstream authentication.
      ProviderAppSecret: ""
      ProviderAppID: ""

      # The cluster ID to delegate the user database.  When set,
      # logins on this cluster will be redirected to the login cluster
      # (login cluster must appear in RemoteHosts with Proxy: true)
      LoginCluster: ""

      # How long a cached token belonging to a remote cluster will
      # remain valid before it needs to be revalidated.
      RemoteTokenRefresh: 5m

    Git:
      # Path to git or gitolite-shell executable. Each authenticated
      # request will execute this program with the single argument "http-backend"
      GitCommand: /usr/bin/git

      # Path to Gitolite's home directory. If a non-empty path is given,
      # the CGI environment will be set up to support the use of
      # gitolite-shell as a GitCommand: for example, if GitoliteHome is
      # "/gh", then the CGI environment will have GITOLITE_HTTP_HOME=/gh,
      # PATH=$PATH:/gh/bin, and GL_BYPASS_ACCESS_CHECKS=1.
      GitoliteHome: ""

      # Git repositories must be readable by api server, or you won't be
      # able to submit crunch jobs. To pass the test suites, put a clone
      # of the arvados tree in {git_repositories_dir}/arvados.git or
      # {git_repositories_dir}/arvados/.git
      Repositories: /var/lib/arvados/git/repositories

    TLS:
      Certificate: ""
      Key: ""
      Insecure: false

    Containers:
      # List of supported Docker Registry image formats that compute nodes
      # are able to use. ` + "`" + `arv keep docker` + "`" + ` will error out if a user tries
      # to store an image with an unsupported format. Use an empty array
      # to skip the compatibility check (and display a warning message to
      # that effect).
      #
      # Example for sites running docker < 1.10: {"v1": {}}
      # Example for sites running docker >= 1.10: {"v2": {}}
      # Example for disabling check: {}
      SupportedDockerImageFormats:
        "v2": {}
        SAMPLE: {}

      # Include details about job reuse decisions in the server log. This
      # causes additional database queries to run, so it should not be
      # enabled unless you expect to examine the resulting logs for
      # troubleshooting purposes.
      LogReuseDecisions: false

      # Default value for keep_cache_ram of a container's runtime_constraints.
      DefaultKeepCacheRAM: 268435456

      # Number of times a container can be unlocked before being
      # automatically cancelled.
      MaxDispatchAttempts: 5

      # Default value for container_count_max for container requests.  This is the
      # number of times Arvados will create a new container to satisfy a container
      # request.  If a container is cancelled it will retry a new container if
      # container_count < container_count_max on any container requests associated
      # with the cancelled container.
      MaxRetryAttempts: 3

      # The maximum number of compute nodes that can be in use simultaneously
      # If this limit is reduced, any existing nodes with slot number >= new limit
      # will not be counted against the new limit. In other words, the new limit
      # won't be strictly enforced until those nodes with higher slot numbers
      # go down.
      MaxComputeVMs: 64

      # Preemptible instance support (e.g. AWS Spot Instances)
      # When true, child containers will get created with the preemptible
      # scheduling parameter parameter set.
      UsePreemptibleInstances: false

      # PEM encoded SSH key (RSA, DSA, or ECDSA) used by the
      # (experimental) cloud dispatcher for executing containers on
      # worker VMs. Begins with "-----BEGIN RSA PRIVATE KEY-----\n"
      # and ends with "\n-----END RSA PRIVATE KEY-----\n".
      DispatchPrivateKey: none

      # Maximum time to wait for workers to come up before abandoning
      # stale locks from a previous dispatch process.
      StaleLockTimeout: 1m

      # The crunch-run command to manage the container on a node
      CrunchRunCommand: "crunch-run"

      # Extra arguments to add to crunch-run invocation
      # Example: ["--cgroup-parent-subsystem=memory"]
      CrunchRunArgumentsList: []

      # Extra RAM to reserve on the node, in addition to
      # the amount specified in the container's RuntimeConstraints
      ReserveExtraRAM: 256MiB

      # Minimum time between two attempts to run the same container
      MinRetryPeriod: 0s

      Logging:
        # When you run the db:delete_old_container_logs task, it will find
        # containers that have been finished for at least this many seconds,
        # and delete their stdout, stderr, arv-mount, crunch-run, and
        # crunchstat logs from the logs table.
        MaxAge: 720h

        # These two settings control how frequently log events are flushed to the
        # database.  Log lines are buffered until either crunch_log_bytes_per_event
        # has been reached or crunch_log_seconds_between_events has elapsed since
        # the last flush.
        LogBytesPerEvent: 4096
        LogSecondsBetweenEvents: 1

        # The sample period for throttling logs.
        LogThrottlePeriod: 60s

        # Maximum number of bytes that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleBytes: 65536

        # Maximum number of lines that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleLines: 1024

        # Maximum bytes that may be logged by a single job.  Log bytes that are
        # silenced by throttling are not counted against this total.
        LimitLogBytesPerJob: 67108864

        LogPartialLineThrottlePeriod: 5s

        # Container logs are written to Keep and saved in a
        # collection, which is updated periodically while the
        # container runs.  This value sets the interval between
        # collection updates.
        LogUpdatePeriod: 30m

        # The log collection is also updated when the specified amount of
        # log data (given in bytes) is produced in less than one update
        # period.
        LogUpdateSize: 32MiB

      SLURM:
        PrioritySpread: 0
        SbatchArgumentsList: []
        SbatchEnvironmentVariables:
          SAMPLE: ""
        Managed:
          # Path to dns server configuration directory
          # (e.g. /etc/unbound.d/conf.d). If false, do not write any config
          # files or touch restart.txt (see below).
          DNSServerConfDir: ""

          # Template file for the dns server host snippets. See
          # unbound.template in this directory for an example. If false, do
          # not write any config files.
          DNSServerConfTemplate: ""

          # String to write to {dns_server_conf_dir}/restart.txt (with a
          # trailing newline) after updating local data. If false, do not
          # open or write the restart.txt file.
          DNSServerReloadCommand: ""

          # Command to run after each DNS update. Template variables will be
          # substituted; see the "unbound" example below. If false, do not run
          # a command.
          DNSServerUpdateCommand: ""

          ComputeNodeDomain: ""
          ComputeNodeNameservers:
            "192.168.1.1": {}
            SAMPLE: {}

          # Hostname to assign to a compute node when it sends a "ping" and the
          # hostname in its Node record is nil.
          # During bootstrapping, the "ping" script is expected to notice the
          # hostname given in the ping response, and update its unix hostname
          # accordingly.
          # If false, leave the hostname alone (this is appropriate if your compute
          # nodes' hostnames are already assigned by some other mechanism).
          #
          # One way or another, the hostnames of your node records should agree
          # with your DNS records and your /etc/slurm-llnl/slurm.conf files.
          #
          # Example for compute0000, compute0001, ....:
          # assign_node_hostname: compute%<slot_number>04d
          # (See http://ruby-doc.org/core-2.2.2/Kernel.html#method-i-format for more.)
          AssignNodeHostname: "compute%<slot_number>d"

      JobsAPI:
        # Enable the legacy 'jobs' API (crunch v1).  This value must be a string.
        #
        # Note: this only enables read-only access, creating new
        # legacy jobs and pipelines is not supported.
        #
        # 'auto' -- (default) enable the Jobs API only if it has been used before
        #         (i.e., there are job records in the database)
        # 'true' -- enable the Jobs API despite lack of existing records.
        # 'false' -- disable the Jobs API despite presence of existing records.
        Enable: 'auto'

        # Git repositories must be readable by api server, or you won't be
        # able to submit crunch jobs. To pass the test suites, put a clone
        # of the arvados tree in {git_repositories_dir}/arvados.git or
        # {git_repositories_dir}/arvados/.git
        GitInternalDir: /var/lib/arvados/internal.git

      CloudVMs:
        # Enable the cloud scheduler (experimental).
        Enable: false

        # Name/number of port where workers' SSH services listen.
        SSHPort: "22"

        # Interval between queue polls.
        PollInterval: 10s

        # Shell command to execute on each worker to determine whether
        # the worker is booted and ready to run containers. It should
        # exit zero if the worker is ready.
        BootProbeCommand: "docker ps -q"

        # Minimum interval between consecutive probes to a single
        # worker.
        ProbeInterval: 10s

        # Maximum probes per second, across all workers in a pool.
        MaxProbesPerSecond: 10

        # Time before repeating SIGTERM when killing a container.
        TimeoutSignal: 5s

        # Time to give up on SIGTERM and write off the worker.
        TimeoutTERM: 2m

        # Maximum create/destroy-instance operations per second (0 =
        # unlimited).
        MaxCloudOpsPerSecond: 0

        # Interval between cloud provider syncs/updates ("list all
        # instances").
        SyncInterval: 1m

        # Time to leave an idle worker running (in case new containers
        # appear in the queue that it can run) before shutting it
        # down.
        TimeoutIdle: 1m

        # Time to wait for a new worker to boot (i.e., pass
        # BootProbeCommand) before giving up and shutting it down.
        TimeoutBooting: 10m

        # Maximum time a worker can stay alive with no successful
        # probes before being automatically shut down.
        TimeoutProbe: 10m

        # Time after shutting down a worker to retry the
        # shutdown/destroy operation.
        TimeoutShutdown: 10s

        # Worker VM image ID.
        ImageID: ""

        # Tags to add on all resources (VMs, NICs, disks) created by
        # the container dispatcher. (Arvados's own tags --
        # InstanceType, IdleBehavior, and InstanceSecret -- will also
        # be added.)
        ResourceTags:
          SAMPLE: "tag value"

        # Prefix for predefined tags used by Arvados (InstanceSetID,
        # InstanceType, InstanceSecret, IdleBehavior). With the
        # default value "Arvados", tags are "ArvadosInstanceSetID",
        # "ArvadosInstanceSecret", etc.
        #
        # This should only be changed while no cloud resources are in
        # use and the cloud dispatcher is not running. Otherwise,
        # VMs/resources that were added using the old tag prefix will
        # need to be detected and cleaned up manually.
        TagKeyPrefix: Arvados

        # Cloud driver: "azure" (Microsoft Azure) or "ec2" (Amazon AWS).
        Driver: ec2

        # Cloud-specific driver parameters.
        DriverParameters:

          # (ec2) Credentials.
          AccessKeyID: ""
          SecretAccessKey: ""

          # (ec2) Instance configuration.
          SecurityGroupIDs:
            "SAMPLE": {}
          SubnetID: ""
          Region: ""
          EBSVolumeType: gp2
          AdminUsername: debian

          # (azure) Credentials.
          SubscriptionID: ""
          ClientID: ""
          ClientSecret: ""
          TenantID: ""

          # (azure) Instance configuration.
          CloudEnvironment: AzurePublicCloud
          ResourceGroup: ""
          Location: centralus
          Network: ""
          Subnet: ""
          StorageAccount: ""
          BlobContainer: ""
          DeleteDanglingResourcesAfter: 20s
          AdminUsername: arvados

    InstanceTypes:

      # Use the instance type name as the key (in place of "SAMPLE" in
      # this sample entry).
      SAMPLE:
        # Cloud provider's instance type. Defaults to the configured type name.
        ProviderType: ""
        VCPUs: 1
        RAM: 128MiB
        IncludedScratch: 16GB
        AddedScratch: 0
        Price: 0.1
        Preemptible: false

    Mail:
      MailchimpAPIKey: ""
      MailchimpListID: ""
      SendUserSetupNotificationEmail: true

      # Bug/issue report notification to and from addresses
      IssueReporterEmailFrom: "arvados@example.com"
      IssueReporterEmailTo: "arvados@example.com"
      SupportEmailAddress: "arvados@example.com"

      # Generic issue email from
      EmailFrom: "arvados@example.com"
    RemoteClusters:
      "*":
        Host: ""
        Proxy: false
        Scheme: https
        Insecure: false
        ActivateUsers: false
      SAMPLE:
        # API endpoint host or host:port; default is {id}.arvadosapi.com
        Host: sample.arvadosapi.com

        # Perform a proxy request when a local client requests an
        # object belonging to this remote.
        Proxy: false

        # Default "https". Can be set to "http" for testing.
        Scheme: https

        # Disable TLS verify. Can be set to true for testing.
        Insecure: false

        # When users present tokens issued by this remote cluster, and
        # their accounts are active on the remote cluster, activate
        # them on this cluster too.
        ActivateUsers: false

    Workbench:
      # Workbench1 configs
      Theme: default
      ActivationContactLink: mailto:info@arvados.org
      ArvadosDocsite: https://doc.arvados.org
      ArvadosPublicDataDocURL: https://playground.arvados.org/projects/public
      ShowUserAgreementInline: false
      SecretKeyBase: ""

      # Scratch directory used by the remote repository browsing
      # feature. If it doesn't exist, it (and any missing parents) will be
      # created using mkdir_p.
      RepositoryCache: /var/www/arvados-workbench/current/tmp/git

      # Below is a sample setting of user_profile_form_fields config parameter.
      # This configuration parameter should be set to either false (to disable) or
      # to a map as shown below.
      # Configure the map of input fields to be displayed in the profile page
      # using the attribute "key" for each of the input fields.
      # This sample shows configuration with one required and one optional form fields.
      # For each of these input fields:
      #   You can specify "Type" as "text" or "select".
      #   List the "Options" to be displayed for each of the "select" menu.
      #   Set "Required" as "true" for any of these fields to make them required.
      # If any of the required fields are missing in the user's profile, the user will be
      # redirected to the profile page before they can access any Workbench features.
      UserProfileFormFields:
        SAMPLE:
          Type: select
          FormFieldTitle: Best color
          FormFieldDescription: your favorite color
          Required: false
          Position: 1
          Options:
            red: {}
            blue: {}
            green: {}
            SAMPLE: {}

        # exampleTextValue:  # key that will be set in properties
        #   Type: text  #
        #   FormFieldTitle: ""
        #   FormFieldDescription: ""
        #   Required: true
        #   Position: 1
        # exampleOptionsValue:
        #   Type: select
        #   FormFieldTitle: ""
        #   FormFieldDescription: ""
        #   Required: true
        #   Position: 1
        #   Options:
        #     red: {}
        #     blue: {}
        #     yellow: {}

      # Use "UserProfileFormMessage to configure the message you want
      # to display on the profile page.
      UserProfileFormMessage: 'Welcome to Arvados. All <span style="color:red">required fields</span> must be completed before you can proceed.'

      # Mimetypes of applications for which the view icon
      # would be enabled in a collection's show page.
      # It is sufficient to list only applications here.
      # No need to list text and image types.
      ApplicationMimetypesWithViewIcon:
        cwl: {}
        fasta: {}
        go: {}
        javascript: {}
        json: {}
        pdf: {}
        python: {}
        x-python: {}
        r: {}
        rtf: {}
        sam: {}
        x-sh: {}
        vnd.realvnc.bed: {}
        xml: {}
        xsl: {}
        SAMPLE: {}

      # The maximum number of bytes to load in the log viewer
      LogViewerMaxBytes: 1M

      # When anonymous_user_token is configured, show public projects page
      EnablePublicProjectsPage: true

      # By default, disable the "Getting Started" popup which is specific to Arvados playground
      EnableGettingStartedPopup: false

      # Ask Arvados API server to compress its response payloads.
      APIResponseCompression: true

      # Timeouts for API requests.
      APIClientConnectTimeout: 2m
      APIClientReceiveTimeout: 5m

      # Maximum number of historic log records of a running job to fetch
      # and display in the Log tab, while subscribing to web sockets.
      RunningJobLogRecordsToFetch: 2000

      # In systems with many shared projects, loading of dashboard and topnav
      # cab be slow due to collections indexing; use the following parameters
      # to suppress these properties
      ShowRecentCollectionsOnDashboard: true
      ShowUserNotifications: true

      # Enable/disable "multi-site search" in top nav ("true"/"false"), or
      # a link to the multi-site search page on a "home" Workbench site.
      #
      # Example:
      #   https://workbench.qr1hi.arvadosapi.com/collections/multisite
      MultiSiteSearch: ""

      # Should workbench allow management of local git repositories? Set to false if
      # the jobs api is disabled and there are no local git repositories.
      Repositories: true

      SiteName: Arvados Workbench
      ProfilingEnabled: false

      # This is related to obsolete Google OpenID 1.0 login
      # but some workbench stuff still expects it to be set.
      DefaultOpenIdPrefix: "https://www.google.com/accounts/o8/id"

      # Workbench2 configs
      VocabularyURL: ""
      FileViewersConfigURL: ""

    # Use experimental controller code (see https://dev.arvados.org/issues/14287)
    EnableBetaController14287: false
`)
