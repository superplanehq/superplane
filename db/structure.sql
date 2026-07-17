--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.10 (Ubuntu 17.10-1.pgdg22.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account_magic_codes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_magic_codes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email character varying(255) NOT NULL,
    code_hash character varying(64) NOT NULL,
    expires_at timestamp without time zone NOT NULL,
    used_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    verify_attempts integer DEFAULT 0 NOT NULL
);


--
-- Name: account_password_auth; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_password_auth (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid NOT NULL,
    password_hash character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: account_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid NOT NULL,
    provider character varying(50) NOT NULL,
    provider_id character varying(255) NOT NULL,
    username character varying(255),
    email character varying(255),
    name character varying(255),
    avatar_url text,
    access_token text,
    refresh_token text,
    token_expires_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    name character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    installation_admin boolean DEFAULT false NOT NULL,
    password_changed_at timestamp with time zone
);


--
-- Name: agent_session_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_session_messages (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    session_id uuid NOT NULL,
    provider_event_id text DEFAULT ''::text NOT NULL,
    role character varying(20) NOT NULL,
    content text DEFAULT ''::text NOT NULL,
    tool_call_id text DEFAULT ''::text NOT NULL,
    tool_name text DEFAULT ''::text NOT NULL,
    tool_status character varying(20) DEFAULT ''::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    images jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: agent_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_sessions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    canvas_id uuid NOT NULL,
    provider character varying(40) NOT NULL,
    provider_session_id text NOT NULL,
    status character varying(40) DEFAULT 'idle'::character varying NOT NULL,
    last_active_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    heartbeat_at timestamp with time zone,
    agent_tool_schema_revision text DEFAULT ''::text NOT NULL,
    context_replayed_at timestamp with time zone,
    tracked_usage_input_tokens bigint DEFAULT 0 NOT NULL,
    tracked_usage_output_tokens bigint DEFAULT 0 NOT NULL,
    tracked_usage_cache_read_tokens bigint DEFAULT 0 NOT NULL,
    tracked_usage_cache_write_tokens bigint DEFAULT 0 NOT NULL,
    tracked_usage_total_tokens bigint DEFAULT 0 NOT NULL,
    tracked_usage_initialized boolean DEFAULT true NOT NULL
);


--
-- Name: app_installation_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    app_installation_id uuid NOT NULL,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    run_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    spec jsonb
);


--
-- Name: app_installation_secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_secrets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    installation_id uuid NOT NULL,
    name character varying(64) NOT NULL,
    value bytea NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    editable boolean DEFAULT false NOT NULL,
    label text,
    description text
);


--
-- Name: app_installation_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installation_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    installation_id uuid NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: app_installations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_installations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    app_name character varying(255) NOT NULL,
    installation_name character varying(255) NOT NULL,
    state character varying(32) NOT NULL,
    state_description character varying(1024),
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    browser_action jsonb,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    deleted_at timestamp with time zone,
    capabilities jsonb DEFAULT '[]'::jsonb NOT NULL,
    properties jsonb DEFAULT '[]'::jsonb NOT NULL,
    setup_state jsonb
);


--
-- Name: app_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.app_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    canvas_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: canvas_folders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvas_folders (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    title character varying(128) NOT NULL,
    background_color character varying(32) DEFAULT 'blue'::character varying NOT NULL,
    sort_order bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    CONSTRAINT canvas_folders_background_color_check CHECK (((background_color)::text = ANY ((ARRAY['blue'::character varying, 'green'::character varying, 'purple'::character varying, 'slate'::character varying, 'orange'::character varying])::text[])))
);


--
-- Name: canvas_memories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvas_memories (
    canvas_id uuid NOT NULL,
    namespace text NOT NULL,
    "values" jsonb NOT NULL,
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    source text DEFAULT 'node'::text NOT NULL
);


--
-- Name: canvas_subscriptions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvas_subscriptions (
    source_canvas_id uuid NOT NULL,
    target_canvas_id uuid NOT NULL,
    target_node_id character varying(128) NOT NULL
);


--
-- Name: casbin_rule; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.casbin_rule (
    id integer NOT NULL,
    ptype character varying(100),
    v0 character varying(100),
    v1 character varying(100),
    v2 character varying(100),
    v3 character varying(100),
    v4 character varying(100),
    v5 character varying(100)
);


--
-- Name: casbin_rule_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.casbin_rule_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: casbin_rule_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.casbin_rule_id_seq OWNED BY public.casbin_rule.id;


--
-- Name: data_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.data_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: email_settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_settings (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    provider character varying(50) NOT NULL,
    smtp_host character varying(255),
    smtp_port integer,
    smtp_username character varying(255),
    smtp_password bytea,
    smtp_from_name character varying(255),
    smtp_from_email character varying(255),
    smtp_use_tls boolean DEFAULT false NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: group_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.group_metadata (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    group_name character varying(255) NOT NULL,
    domain_type character varying(50) NOT NULL,
    domain_id character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: installation_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.installation_metadata (
    id integer NOT NULL,
    installation_id character varying(64) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    allow_private_network_access boolean DEFAULT false NOT NULL,
    signups_enabled boolean DEFAULT true NOT NULL,
    CONSTRAINT installation_metadata_singleton CHECK ((id = 1))
);


--
-- Name: organization_invitations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_invitations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    email character varying(255) NOT NULL,
    invited_by uuid NOT NULL,
    state character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    canvas_ids jsonb DEFAULT '[]'::jsonb
);


--
-- Name: organization_invite_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organization_invite_links (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    token uuid NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    allowed_providers jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    description text DEFAULT ''::text,
    usage_synced_at timestamp with time zone,
    usage_retention_window_days integer,
    usage_limits_synced_at timestamp with time zone,
    enabled_experimental_features jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: repositories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repositories (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    provider text NOT NULL,
    repo_id text NOT NULL,
    status character varying(64) DEFAULT 'pending'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: repository_seed_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.repository_seed_files (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    repository_id uuid NOT NULL,
    path text NOT NULL,
    content bytea NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: role_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.role_metadata (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    role_name character varying(255) NOT NULL,
    domain_type character varying(50) NOT NULL,
    domain_id character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: secrets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.secrets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    provider character varying(64) NOT NULL,
    data bytea NOT NULL,
    domain_type character varying(64) NOT NULL,
    domain_id character varying(64) NOT NULL
);


--
-- Name: user_canvas_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_canvas_preferences (
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    canvas_id uuid NOT NULL,
    starred_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    account_id uuid,
    name character varying(255),
    email character varying(255),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    organization_id uuid NOT NULL,
    token_hash character varying(250),
    type character varying(50) DEFAULT 'human'::character varying NOT NULL,
    description text,
    created_by uuid,
    api_key_expires_at timestamp without time zone,
    api_key_canvas_ids jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: webhooks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.webhooks (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    state character varying(32) NOT NULL,
    secret bytea NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    deleted_at timestamp without time zone,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    app_installation_id uuid
);


--
-- Name: workflow_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128),
    channel character varying(64),
    data jsonb NOT NULL,
    state character varying(32) NOT NULL,
    execution_id uuid,
    created_at timestamp without time zone NOT NULL,
    custom_name text,
    run_id uuid NOT NULL
);


--
-- Name: workflow_node_execution_kvs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_execution_kvs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    execution_id uuid NOT NULL,
    key text NOT NULL,
    value text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL
);


--
-- Name: workflow_node_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_executions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid,
    event_id uuid,
    previous_execution_id uuid,
    state character varying(32) NOT NULL,
    result character varying(32),
    result_reason character varying(128),
    result_message text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    cancelled_by uuid,
    run_id uuid NOT NULL,
    cancelled_at timestamp without time zone
);


--
-- Name: workflow_node_queue_items; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_queue_items (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    root_event_id uuid,
    event_id uuid,
    created_at timestamp without time zone NOT NULL,
    run_id uuid NOT NULL
);


--
-- Name: workflow_node_requests; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_node_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    execution_id uuid,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    spec jsonb NOT NULL,
    run_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    node_id character varying(128) NOT NULL
);


--
-- Name: workflow_nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_nodes (
    workflow_id uuid NOT NULL,
    node_id character varying(128) NOT NULL,
    name character varying(128) NOT NULL,
    state character varying(32) NOT NULL,
    type character varying(32) NOT NULL,
    ref jsonb NOT NULL,
    configuration jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    webhook_id uuid,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    "position" jsonb DEFAULT '{}'::jsonb NOT NULL,
    is_collapsed boolean DEFAULT false NOT NULL,
    deleted_at timestamp with time zone,
    app_installation_id uuid,
    state_reason text
);


--
-- Name: workflow_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_runs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    state character varying(32) NOT NULL,
    result character varying(32),
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    finished_at timestamp without time zone,
    version_id uuid NOT NULL,
    cancelled_at timestamp without time zone,
    cancelled_by uuid,
    parent_run_id uuid,
    parent_workflow_id uuid,
    parent_execution_id uuid,
    callbacks jsonb DEFAULT '[]'::jsonb NOT NULL,
    input jsonb DEFAULT '{}'::jsonb NOT NULL,
    node_id character varying(255)
);


--
-- Name: workflow_staged_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_staged_files (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    base_version_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    path text NOT NULL,
    content text DEFAULT ''::text NOT NULL,
    deleted boolean DEFAULT false NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_id uuid NOT NULL,
    workflow_id uuid NOT NULL
);


--
-- Name: workflow_versions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflow_versions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    workflow_id uuid NOT NULL,
    owner_id uuid,
    nodes jsonb DEFAULT '[]'::jsonb NOT NULL,
    edges jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    console_panels jsonb DEFAULT '[]'::jsonb NOT NULL,
    console_layout jsonb DEFAULT '[]'::jsonb NOT NULL,
    commit_sha character varying(40) DEFAULT ''::character varying NOT NULL,
    commit_message text DEFAULT ''::text NOT NULL
);


--
-- Name: workflows; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.workflows (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_by uuid,
    deleted_at timestamp without time zone,
    live_version_id uuid NOT NULL,
    folder_id uuid,
    description text DEFAULT ''::text NOT NULL
);


--
-- Name: casbin_rule id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule ALTER COLUMN id SET DEFAULT nextval('public.casbin_rule_id_seq'::regclass);


--
-- Name: account_magic_codes account_magic_codes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_magic_codes
    ADD CONSTRAINT account_magic_codes_pkey PRIMARY KEY (id);


--
-- Name: account_password_auth account_password_auth_account_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_account_id_key UNIQUE (account_id);


--
-- Name: account_password_auth account_password_auth_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_pkey PRIMARY KEY (id);


--
-- Name: account_providers account_providers_account_id_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_account_id_provider_key UNIQUE (account_id, provider);


--
-- Name: account_providers account_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_pkey PRIMARY KEY (id);


--
-- Name: account_providers account_providers_provider_provider_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_provider_provider_id_key UNIQUE (provider, provider_id);


--
-- Name: accounts accounts_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_email_key UNIQUE (email);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: agent_session_messages agent_session_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_session_messages
    ADD CONSTRAINT agent_session_messages_pkey PRIMARY KEY (id);


--
-- Name: agent_sessions agent_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_sessions
    ADD CONSTRAINT agent_sessions_pkey PRIMARY KEY (id);


--
-- Name: app_installation_requests app_installation_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_requests
    ADD CONSTRAINT app_installation_requests_pkey PRIMARY KEY (id);


--
-- Name: app_installation_secrets app_installation_secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_pkey PRIMARY KEY (id);


--
-- Name: app_installation_subscriptions app_installation_subscription_installation_id_workflow_id_n_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscription_installation_id_workflow_id_n_key UNIQUE (installation_id, workflow_id, node_id);


--
-- Name: app_installation_subscriptions app_installation_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: app_installations app_installations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installations
    ADD CONSTRAINT app_installations_pkey PRIMARY KEY (id);


--
-- Name: app_messages app_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_messages
    ADD CONSTRAINT app_messages_pkey PRIMARY KEY (id);


--
-- Name: canvas_folders canvas_folders_organization_id_title_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_folders
    ADD CONSTRAINT canvas_folders_organization_id_title_key UNIQUE (organization_id, title);


--
-- Name: canvas_folders canvas_folders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_folders
    ADD CONSTRAINT canvas_folders_pkey PRIMARY KEY (id);


--
-- Name: canvas_memories canvas_memories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_memories
    ADD CONSTRAINT canvas_memories_pkey PRIMARY KEY (id);


--
-- Name: canvas_subscriptions canvas_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_subscriptions
    ADD CONSTRAINT canvas_subscriptions_pkey PRIMARY KEY (source_canvas_id, target_canvas_id, target_node_id);


--
-- Name: casbin_rule casbin_rule_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule
    ADD CONSTRAINT casbin_rule_pkey PRIMARY KEY (id);


--
-- Name: data_migrations data_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.data_migrations
    ADD CONSTRAINT data_migrations_pkey PRIMARY KEY (version);


--
-- Name: email_settings email_settings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_settings
    ADD CONSTRAINT email_settings_pkey PRIMARY KEY (id);


--
-- Name: email_settings email_settings_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_settings
    ADD CONSTRAINT email_settings_provider_key UNIQUE (provider);


--
-- Name: group_metadata group_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_metadata
    ADD CONSTRAINT group_metadata_pkey PRIMARY KEY (id);


--
-- Name: installation_metadata installation_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.installation_metadata
    ADD CONSTRAINT installation_metadata_pkey PRIMARY KEY (id);


--
-- Name: organization_invitations organization_invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_pkey PRIMARY KEY (id);


--
-- Name: organization_invite_links organization_invite_links_organization_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_key UNIQUE (organization_id);


--
-- Name: organization_invite_links organization_invite_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_pkey PRIMARY KEY (id);


--
-- Name: organization_invite_links organization_invite_links_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_token_key UNIQUE (token);


--
-- Name: organizations organizations_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_name_key UNIQUE (name);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: repositories repositories_canvas_id_provider_repo_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories
    ADD CONSTRAINT repositories_canvas_id_provider_repo_id_key UNIQUE (canvas_id, provider, repo_id);


--
-- Name: repositories repositories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories
    ADD CONSTRAINT repositories_pkey PRIMARY KEY (id);


--
-- Name: repository_seed_files repository_seed_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_seed_files
    ADD CONSTRAINT repository_seed_files_pkey PRIMARY KEY (id);


--
-- Name: repository_seed_files repository_seed_files_repository_id_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_seed_files
    ADD CONSTRAINT repository_seed_files_repository_id_path_key UNIQUE (repository_id, path);


--
-- Name: role_metadata role_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_metadata
    ADD CONSTRAINT role_metadata_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: secrets secrets_domain_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_domain_id_name_key UNIQUE (domain_type, domain_id, name);


--
-- Name: secrets secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);


--
-- Name: group_metadata uq_group_metadata_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.group_metadata
    ADD CONSTRAINT uq_group_metadata_key UNIQUE (group_name, domain_type, domain_id);


--
-- Name: role_metadata uq_role_metadata_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.role_metadata
    ADD CONSTRAINT uq_role_metadata_key UNIQUE (role_name, domain_type, domain_id);


--
-- Name: user_canvas_preferences user_canvas_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_canvas_preferences
    ADD CONSTRAINT user_canvas_preferences_pkey PRIMARY KEY (organization_id, user_id, canvas_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: webhooks webhooks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_pkey PRIMARY KEY (id);


--
-- Name: workflow_events workflow_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_execution_kvs workflow_node_execution_kvs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT workflow_node_execution_kvs_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_requests workflow_node_execution_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT workflow_node_execution_requests_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_executions workflow_node_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_pkey PRIMARY KEY (id);


--
-- Name: workflow_node_queue_items workflow_node_queue_items_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_pkey PRIMARY KEY (id);


--
-- Name: workflow_nodes workflow_nodes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_pkey PRIMARY KEY (workflow_id, node_id);


--
-- Name: workflow_runs workflow_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_pkey PRIMARY KEY (id);


--
-- Name: workflow_staged_files workflow_staged_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_pkey PRIMARY KEY (id);


--
-- Name: workflow_staged_files workflow_staged_files_workflow_user_path_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_workflow_user_path_key UNIQUE (workflow_id, user_id, path);


--
-- Name: workflow_versions workflow_versions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_pkey PRIMARY KEY (id);


--
-- Name: workflows workflows_organization_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: workflows workflows_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_pkey PRIMARY KEY (id);


--
-- Name: agent_session_messages_provider_event_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX agent_session_messages_provider_event_idx ON public.agent_session_messages USING btree (session_id, provider_event_id) WHERE (provider_event_id <> ''::text);


--
-- Name: agent_session_messages_session_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX agent_session_messages_session_idx ON public.agent_session_messages USING btree (session_id, created_at DESC, id DESC);


--
-- Name: agent_sessions_provider_session_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX agent_sessions_provider_session_id_idx ON public.agent_sessions USING btree (provider, provider_session_id);


--
-- Name: agent_sessions_user_canvas_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX agent_sessions_user_canvas_idx ON public.agent_sessions USING btree (organization_id, user_id, canvas_id);


--
-- Name: idx_account_magic_codes_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_magic_codes_email ON public.account_magic_codes USING btree (email);


--
-- Name: idx_account_magic_codes_email_code_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_magic_codes_email_code_hash ON public.account_magic_codes USING btree (email, code_hash);


--
-- Name: idx_account_password_auth_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_password_auth_account_id ON public.account_password_auth USING btree (account_id);


--
-- Name: idx_account_providers_account_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_account_id ON public.account_providers USING btree (account_id);


--
-- Name: idx_account_providers_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_provider ON public.account_providers USING btree (provider);


--
-- Name: idx_app_installation_requests_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_requests_installation_id ON public.app_installation_requests USING btree (app_installation_id);


--
-- Name: idx_app_installation_requests_state_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_requests_state_run_at ON public.app_installation_requests USING btree (state, run_at) WHERE ((state)::text = 'pending'::text);


--
-- Name: idx_app_installation_secrets_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_secrets_installation_id ON public.app_installation_secrets USING btree (installation_id);


--
-- Name: idx_app_installation_secrets_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_secrets_organization_id ON public.app_installation_secrets USING btree (organization_id);


--
-- Name: idx_app_installation_subscriptions_installation; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_installation ON public.app_installation_subscriptions USING btree (installation_id);


--
-- Name: idx_app_installation_subscriptions_node; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_node ON public.app_installation_subscriptions USING btree (workflow_id, node_id);


--
-- Name: idx_app_installation_subscriptions_workflow; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installation_subscriptions_workflow ON public.app_installation_subscriptions USING btree (workflow_id);


--
-- Name: idx_app_installations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installations_deleted_at ON public.app_installations USING btree (deleted_at);


--
-- Name: idx_app_installations_org_name_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_app_installations_org_name_unique ON public.app_installations USING btree (organization_id, installation_name);


--
-- Name: idx_app_installations_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_installations_organization_id ON public.app_installations USING btree (organization_id);


--
-- Name: idx_app_messages_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_app_messages_created_at ON public.app_messages USING btree (created_at);


--
-- Name: idx_canvas_folders_organization_id_title; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_canvas_folders_organization_id_title ON public.canvas_folders USING btree (organization_id, title);


--
-- Name: idx_canvas_memories_canvas_namespace; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_canvas_memories_canvas_namespace ON public.canvas_memories USING btree (canvas_id, namespace);


--
-- Name: idx_canvas_subscriptions_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_canvas_subscriptions_target ON public.canvas_subscriptions USING btree (target_canvas_id, target_node_id);


--
-- Name: idx_casbin_rule; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_casbin_rule ON public.casbin_rule USING btree (ptype, v0, v1, v2, v3, v4, v5);


--
-- Name: idx_casbin_rule_ptype; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_ptype ON public.casbin_rule USING btree (ptype);


--
-- Name: idx_casbin_rule_v0; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v0 ON public.casbin_rule USING btree (v0);


--
-- Name: idx_casbin_rule_v1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v1 ON public.casbin_rule USING btree (v1);


--
-- Name: idx_casbin_rule_v2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_casbin_rule_v2 ON public.casbin_rule USING btree (v2);


--
-- Name: idx_group_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_group_metadata_lookup ON public.group_metadata USING btree (group_name, domain_type, domain_id);


--
-- Name: idx_node_requests_state_run_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_node_requests_state_run_at ON public.workflow_node_requests USING btree (state, run_at) WHERE ((state)::text = 'pending'::text);


--
-- Name: idx_organizations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizations_deleted_at ON public.organizations USING btree (deleted_at);


--
-- Name: idx_repositories_canvas_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repositories_canvas_id ON public.repositories USING btree (canvas_id);


--
-- Name: idx_repository_seed_files_repository_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_repository_seed_files_repository_id ON public.repository_seed_files USING btree (repository_id);


--
-- Name: idx_role_metadata_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_role_metadata_lookup ON public.role_metadata USING btree (role_name, domain_type, domain_id);


--
-- Name: idx_user_canvas_preferences_user_starred; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_canvas_preferences_user_starred ON public.user_canvas_preferences USING btree (organization_id, user_id, starred_at DESC) WHERE (starred_at IS NOT NULL);


--
-- Name: idx_webhooks_app_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhooks_app_installation_id ON public.webhooks USING btree (app_installation_id);


--
-- Name: idx_webhooks_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_webhooks_deleted_at ON public.webhooks USING btree (deleted_at);


--
-- Name: idx_workflow_events_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_execution_id ON public.workflow_events USING btree (execution_id);


--
-- Name: idx_workflow_events_run_id_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_run_id_state ON public.workflow_events USING btree (run_id, state);


--
-- Name: idx_workflow_events_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_state ON public.workflow_events USING btree (state);


--
-- Name: idx_workflow_events_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_events_workflow_node_id ON public.workflow_events USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_node_execution_kvs_ekv; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_execution_kvs_ekv ON public.workflow_node_execution_kvs USING btree (execution_id, key, value);


--
-- Name: idx_workflow_node_execution_kvs_workflow_node_key_value; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_execution_kvs_workflow_node_key_value ON public.workflow_node_execution_kvs USING btree (workflow_id, node_id, key, value);


--
-- Name: idx_workflow_node_executions_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_event_id ON public.workflow_node_executions USING btree (event_id);


--
-- Name: idx_workflow_node_executions_previous_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_previous_execution_id ON public.workflow_node_executions USING btree (previous_execution_id);


--
-- Name: idx_workflow_node_executions_root_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_root_event_id ON public.workflow_node_executions USING btree (root_event_id);


--
-- Name: idx_workflow_node_executions_run_id_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_run_id_state ON public.workflow_node_executions USING btree (run_id, state);


--
-- Name: idx_workflow_node_executions_state_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_state_created_at ON public.workflow_node_executions USING btree (state, created_at DESC);


--
-- Name: idx_workflow_node_executions_workflow_node_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_executions_workflow_node_id ON public.workflow_node_executions USING btree (workflow_id, node_id);


--
-- Name: idx_workflow_node_installation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_installation_id ON public.workflow_nodes USING btree (app_installation_id);


--
-- Name: idx_workflow_node_queue_items_root_event_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_queue_items_root_event_id ON public.workflow_node_queue_items USING btree (root_event_id);


--
-- Name: idx_workflow_node_queue_items_run_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_queue_items_run_id ON public.workflow_node_queue_items USING btree (run_id);


--
-- Name: idx_workflow_node_requests_execution_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_node_requests_execution_id ON public.workflow_node_requests USING btree (execution_id);


--
-- Name: idx_workflow_nodes_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_deleted_at ON public.workflow_nodes USING btree (deleted_at);


--
-- Name: idx_workflow_nodes_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_nodes_state ON public.workflow_nodes USING btree (state);


--
-- Name: idx_workflow_runs_cancelling; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_runs_cancelling ON public.workflow_runs USING btree (cancelled_at) WHERE ((state)::text = 'cancelling'::text);


--
-- Name: idx_workflow_runs_version_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_runs_version_id ON public.workflow_runs USING btree (version_id);


--
-- Name: idx_workflow_runs_workflow_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_runs_workflow_created_at ON public.workflow_runs USING btree (workflow_id, created_at DESC);


--
-- Name: idx_workflow_runs_workflow_state; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_runs_workflow_state ON public.workflow_runs USING btree (workflow_id, state);


--
-- Name: idx_workflow_staged_files_workflow_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_staged_files_workflow_user ON public.workflow_staged_files USING btree (workflow_id, user_id);


--
-- Name: idx_workflow_versions_commit_sha; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_versions_commit_sha ON public.workflow_versions USING btree (workflow_id, commit_sha) WHERE ((commit_sha)::text <> ''::text);


--
-- Name: idx_workflow_versions_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_versions_owner ON public.workflow_versions USING btree (owner_id);


--
-- Name: idx_workflow_versions_workflow_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflow_versions_workflow_id ON public.workflow_versions USING btree (workflow_id);


--
-- Name: idx_workflows_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_deleted_at ON public.workflows USING btree (deleted_at);


--
-- Name: idx_workflows_folder_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_folder_id ON public.workflows USING btree (folder_id);


--
-- Name: idx_workflows_live_version_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_live_version_id ON public.workflows USING btree (live_version_id);


--
-- Name: idx_workflows_organization_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_workflows_organization_id ON public.workflows USING btree (organization_id);


--
-- Name: unique_api_key_in_organization; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_api_key_in_organization ON public.users USING btree (organization_id, name) WHERE ((type)::text = 'api_key'::text);


--
-- Name: unique_human_user_in_organization; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX unique_human_user_in_organization ON public.users USING btree (organization_id, account_id, email) WHERE ((type)::text = 'human'::text);


--
-- Name: account_password_auth account_password_auth_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_password_auth
    ADD CONSTRAINT account_password_auth_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id) ON DELETE CASCADE;


--
-- Name: account_providers account_providers_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: agent_session_messages agent_session_messages_session_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_session_messages
    ADD CONSTRAINT agent_session_messages_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.agent_sessions(id) ON DELETE CASCADE;


--
-- Name: app_installation_requests app_installation_requests_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_requests
    ADD CONSTRAINT app_installation_requests_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_secrets app_installation_secrets_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_secrets app_installation_secrets_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_secrets
    ADD CONSTRAINT app_installation_secrets_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_installation_id_fkey FOREIGN KEY (installation_id) REFERENCES public.app_installations(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: app_installation_subscriptions app_installation_subscriptions_workflow_id_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installation_subscriptions
    ADD CONSTRAINT app_installation_subscriptions_workflow_id_node_id_fkey FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;


--
-- Name: app_installations app_installations_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_installations
    ADD CONSTRAINT app_installations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: app_messages app_messages_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_messages
    ADD CONSTRAINT app_messages_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: app_messages app_messages_canvas_id_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.app_messages
    ADD CONSTRAINT app_messages_canvas_id_node_id_fkey FOREIGN KEY (canvas_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;


--
-- Name: canvas_folders canvas_folders_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_folders
    ADD CONSTRAINT canvas_folders_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: canvas_memories canvas_memories_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_memories
    ADD CONSTRAINT canvas_memories_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: canvas_subscriptions canvas_subscriptions_source_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_subscriptions
    ADD CONSTRAINT canvas_subscriptions_source_canvas_id_fkey FOREIGN KEY (source_canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: canvas_subscriptions canvas_subscriptions_target_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_subscriptions
    ADD CONSTRAINT canvas_subscriptions_target_canvas_id_fkey FOREIGN KEY (target_canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: canvas_subscriptions canvas_subscriptions_target_canvas_id_target_node_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvas_subscriptions
    ADD CONSTRAINT canvas_subscriptions_target_canvas_id_target_node_id_fkey FOREIGN KEY (target_canvas_id, target_node_id) REFERENCES public.workflow_nodes(workflow_id, node_id) ON DELETE CASCADE;


--
-- Name: workflow_node_execution_kvs fk_wnek_workflow; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_execution_kvs fk_wnek_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT fk_wnek_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_events fk_workflow_events_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT fk_workflow_events_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_executions fk_workflow_node_executions_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT fk_workflow_node_executions_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_queue_items fk_workflow_node_queue_items_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT fk_workflow_node_queue_items_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: workflow_node_requests fk_workflow_node_requests_workflow_node; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT fk_workflow_node_requests_workflow_node FOREIGN KEY (workflow_id, node_id) REFERENCES public.workflow_nodes(workflow_id, node_id);


--
-- Name: organization_invitations organization_invitations_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invitations
    ADD CONSTRAINT organization_invitations_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: organization_invite_links organization_invite_links_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organization_invite_links
    ADD CONSTRAINT organization_invite_links_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: repositories repositories_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories
    ADD CONSTRAINT repositories_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: repositories repositories_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repositories
    ADD CONSTRAINT repositories_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: repository_seed_files repository_seed_files_repository_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.repository_seed_files
    ADD CONSTRAINT repository_seed_files_repository_id_fkey FOREIGN KEY (repository_id) REFERENCES public.repositories(id) ON DELETE CASCADE;


--
-- Name: user_canvas_preferences user_canvas_preferences_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_canvas_preferences
    ADD CONSTRAINT user_canvas_preferences_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: user_canvas_preferences user_canvas_preferences_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_canvas_preferences
    ADD CONSTRAINT user_canvas_preferences_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: user_canvas_preferences user_canvas_preferences_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_canvas_preferences
    ADD CONSTRAINT user_canvas_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: users users_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id);


--
-- Name: users users_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: webhooks webhooks_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.webhooks
    ADD CONSTRAINT webhooks_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id);


--
-- Name: workflow_events workflow_events_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_events workflow_events_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.workflow_runs(id);


--
-- Name: workflow_events workflow_events_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_events
    ADD CONSTRAINT workflow_events_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_execution_kvs workflow_node_execution_kvs_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_execution_kvs
    ADD CONSTRAINT workflow_node_execution_kvs_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_executions workflow_node_executions_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_previous_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_previous_execution_id_fkey FOREIGN KEY (previous_execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_executions workflow_node_executions_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.workflow_runs(id);


--
-- Name: workflow_node_executions workflow_node_executions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_executions
    ADD CONSTRAINT workflow_node_executions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_queue_items workflow_node_queue_items_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_root_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_root_event_id_fkey FOREIGN KEY (root_event_id) REFERENCES public.workflow_events(id) ON DELETE SET NULL;


--
-- Name: workflow_node_queue_items workflow_node_queue_items_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_run_id_fkey FOREIGN KEY (run_id) REFERENCES public.workflow_runs(id);


--
-- Name: workflow_node_queue_items workflow_node_queue_items_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_queue_items
    ADD CONSTRAINT workflow_node_queue_items_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_node_requests workflow_node_requests_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.workflow_node_executions(id) ON DELETE CASCADE;


--
-- Name: workflow_node_requests workflow_node_requests_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_node_requests
    ADD CONSTRAINT workflow_node_requests_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_nodes workflow_nodes_app_installation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_app_installation_id_fkey FOREIGN KEY (app_installation_id) REFERENCES public.app_installations(id) ON DELETE SET NULL;


--
-- Name: workflow_nodes workflow_nodes_webhook_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_webhook_id_fkey FOREIGN KEY (webhook_id) REFERENCES public.webhooks(id) ON DELETE SET NULL;


--
-- Name: workflow_nodes workflow_nodes_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_nodes
    ADD CONSTRAINT workflow_nodes_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_runs workflow_runs_cancelled_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_cancelled_by_fkey FOREIGN KEY (cancelled_by) REFERENCES public.users(id);


--
-- Name: workflow_runs workflow_runs_parent_execution_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_parent_execution_id_fkey FOREIGN KEY (parent_execution_id) REFERENCES public.workflow_node_executions(id);


--
-- Name: workflow_runs workflow_runs_parent_run_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_parent_run_id_fkey FOREIGN KEY (parent_run_id) REFERENCES public.workflow_runs(id);


--
-- Name: workflow_runs workflow_runs_parent_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_parent_workflow_id_fkey FOREIGN KEY (parent_workflow_id) REFERENCES public.workflows(id);


--
-- Name: workflow_runs workflow_runs_version_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_version_id_fkey FOREIGN KEY (version_id) REFERENCES public.workflow_versions(id) ON DELETE RESTRICT;


--
-- Name: workflow_runs workflow_runs_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_runs
    ADD CONSTRAINT workflow_runs_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_staged_files workflow_staged_files_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: workflow_staged_files workflow_staged_files_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: workflow_staged_files workflow_staged_files_version_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_version_id_fkey FOREIGN KEY (base_version_id) REFERENCES public.workflow_versions(id) ON DELETE CASCADE;


--
-- Name: workflow_staged_files workflow_staged_files_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_staged_files
    ADD CONSTRAINT workflow_staged_files_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflow_versions workflow_versions_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: workflow_versions workflow_versions_workflow_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflow_versions
    ADD CONSTRAINT workflow_versions_workflow_id_fkey FOREIGN KEY (workflow_id) REFERENCES public.workflows(id) ON DELETE CASCADE;


--
-- Name: workflows workflows_folder_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_folder_id_fkey FOREIGN KEY (folder_id) REFERENCES public.canvas_folders(id) ON DELETE SET NULL;


--
-- Name: workflows workflows_live_version_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.workflows
    ADD CONSTRAINT workflows_live_version_id_fkey FOREIGN KEY (live_version_id) REFERENCES public.workflow_versions(id) ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.10 (Ubuntu 17.10-1.pgdg22.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.schema_migrations (version, dirty) FROM stdin;
20260717191931	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.10 (Ubuntu 17.10-1.pgdg22.04+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: data_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.data_migrations (version, dirty) FROM stdin;
20260709012138	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

