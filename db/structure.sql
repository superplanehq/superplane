--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg120+1)
-- Dumped by pg_dump version 17.5 (Debian 17.5-1.pgdg120+1)

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
-- Name: account_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account_providers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
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
-- Name: canvases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.canvases (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    organization_id uuid NOT NULL
);


--
-- Name: casbin_rule; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.casbin_rule (
    id integer NOT NULL,
    ptype character varying(100) NOT NULL,
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
-- Name: event_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.event_sources (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    canvas_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    key bytea NOT NULL
);


--
-- Name: events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    received_at timestamp without time zone NOT NULL,
    raw jsonb NOT NULL,
    state character varying(64) NOT NULL,
    headers jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    display_name character varying(255) NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone
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
    canvas_id uuid NOT NULL,
    name character varying(128) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    provider character varying(64) NOT NULL,
    data bytea NOT NULL
);


--
-- Name: stage_connections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    filter_operator character varying(16) NOT NULL,
    filters jsonb NOT NULL
);


--
-- Name: stage_event_approvals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_event_approvals (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_event_id uuid NOT NULL,
    approved_at timestamp without time zone NOT NULL,
    approved_by uuid NOT NULL
);


--
-- Name: stage_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_id uuid NOT NULL,
    event_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_name character varying(128) NOT NULL,
    source_type character varying(64) NOT NULL,
    state character varying(64) NOT NULL,
    state_reason character varying(64),
    created_at timestamp without time zone NOT NULL,
    inputs jsonb DEFAULT '{}'::jsonb NOT NULL
);


--
-- Name: stage_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stage_executions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    stage_id uuid NOT NULL,
    stage_event_id uuid NOT NULL,
    reference_id character varying(64) NOT NULL,
    state character varying(64) NOT NULL,
    result character varying(64) NOT NULL,
    outputs jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    started_at timestamp without time zone,
    finished_at timestamp without time zone
);


--
-- Name: stages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stages (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(128) NOT NULL,
    canvas_id uuid NOT NULL,
    created_at timestamp without time zone NOT NULL,
    created_by uuid NOT NULL,
    updated_at timestamp without time zone,
    updated_by uuid,
    executor_spec jsonb NOT NULL,
    conditions jsonb,
    inputs jsonb DEFAULT '[]'::jsonb NOT NULL,
    outputs jsonb DEFAULT '[]'::jsonb NOT NULL,
    input_mappings jsonb DEFAULT '[]'::jsonb NOT NULL,
    secrets jsonb DEFAULT '[]'::jsonb NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: casbin_rule id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule ALTER COLUMN id SET DEFAULT nextval('public.casbin_rule_id_seq'::regclass);


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
-- Name: account_providers account_providers_user_id_provider_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_user_id_provider_key UNIQUE (user_id, provider);


--
-- Name: canvases canvases_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT canvases_name_key UNIQUE (name);


--
-- Name: canvases canvases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT canvases_pkey PRIMARY KEY (id);


--
-- Name: casbin_rule casbin_rule_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.casbin_rule
    ADD CONSTRAINT casbin_rule_pkey PRIMARY KEY (id);


--
-- Name: event_sources event_sources_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: event_sources event_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


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
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: secrets secrets_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: secrets secrets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);


--
-- Name: stage_connections stage_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_connections
    ADD CONSTRAINT stage_connections_pkey PRIMARY KEY (id);


--
-- Name: stage_connections stage_connections_stage_id_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_connections
    ADD CONSTRAINT stage_connections_stage_id_source_id_key UNIQUE (stage_id, source_id);


--
-- Name: stage_event_approvals stage_event_approvals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_pkey PRIMARY KEY (id);


--
-- Name: stage_event_approvals stage_event_approvals_stage_event_id_approved_by_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_stage_event_id_approved_by_key UNIQUE (stage_event_id, approved_by);


--
-- Name: stage_events stage_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_events
    ADD CONSTRAINT stage_events_pkey PRIMARY KEY (id);


--
-- Name: stage_executions stage_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_pkey PRIMARY KEY (id);


--
-- Name: stages stages_canvas_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_canvas_id_name_key UNIQUE (canvas_id, name);


--
-- Name: stages stages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_account_providers_provider; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_provider ON public.account_providers USING btree (provider);


--
-- Name: idx_account_providers_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_providers_user_id ON public.account_providers USING btree (user_id);


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
-- Name: idx_organizations_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_organizations_deleted_at ON public.organizations USING btree (deleted_at);


--
-- Name: uix_event_sources_canvas; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_event_sources_canvas ON public.event_sources USING btree (canvas_id);


--
-- Name: uix_events_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_events_source ON public.events USING btree (source_id);


--
-- Name: uix_stage_connections_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_connections_stage ON public.stage_connections USING btree (stage_id);


--
-- Name: uix_stage_event_approvals_events; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_event_approvals_events ON public.stage_event_approvals USING btree (stage_event_id);


--
-- Name: uix_stage_events_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_events_source ON public.stage_events USING btree (source_id);


--
-- Name: uix_stage_events_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_events_stage ON public.stage_events USING btree (stage_id);


--
-- Name: uix_stage_executions_events; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_executions_events ON public.stage_executions USING btree (stage_event_id);


--
-- Name: uix_stage_executions_stage; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stage_executions_stage ON public.stage_executions USING btree (stage_id);


--
-- Name: uix_stages_canvas; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX uix_stages_canvas ON public.stages USING btree (canvas_id);


--
-- Name: account_providers account_providers_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account_providers
    ADD CONSTRAINT account_providers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: canvases canvases_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.canvases
    ADD CONSTRAINT canvases_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: event_sources event_sources_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.event_sources
    ADD CONSTRAINT event_sources_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: secrets secrets_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- Name: stage_connections stage_connections_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_connections
    ADD CONSTRAINT stage_connections_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stages(id);


--
-- Name: stage_event_approvals stage_event_approvals_stage_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_event_approvals
    ADD CONSTRAINT stage_event_approvals_stage_event_id_fkey FOREIGN KEY (stage_event_id) REFERENCES public.stage_events(id);


--
-- Name: stage_events stage_events_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_events
    ADD CONSTRAINT stage_events_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stages(id);


--
-- Name: stage_executions stage_executions_stage_event_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_stage_event_id_fkey FOREIGN KEY (stage_event_id) REFERENCES public.stage_events(id);


--
-- Name: stage_executions stage_executions_stage_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stage_executions
    ADD CONSTRAINT stage_executions_stage_id_fkey FOREIGN KEY (stage_id) REFERENCES public.stages(id);


--
-- Name: stages stages_canvas_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stages
    ADD CONSTRAINT stages_canvas_id_fkey FOREIGN KEY (canvas_id) REFERENCES public.canvases(id);


--
-- PostgreSQL database dump complete
--

--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg120+1)
-- Dumped by pg_dump version 17.5 (Debian 17.5-1.pgdg120+1)

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
20250618171050	f
\.


--
-- PostgreSQL database dump complete
--

