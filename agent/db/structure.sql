--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.9 (Ubuntu 17.9-1.pgdg22.04+1)

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: agent_chat_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_chat_messages (
    id uuid NOT NULL,
    chat_id uuid NOT NULL,
    message_index integer NOT NULL,
    message jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: agent_chats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_chats (
    id uuid NOT NULL,
    org_id uuid NOT NULL,
    user_id uuid NOT NULL,
    canvas_id uuid NOT NULL,
    initial_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: agent_chat_messages agent_chat_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_chat_messages
    ADD CONSTRAINT agent_chat_messages_pkey PRIMARY KEY (id);


--
-- Name: agent_chats agent_chats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_chats
    ADD CONSTRAINT agent_chats_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: idx_agent_chat_messages_chat_id_message_index; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_agent_chat_messages_chat_id_message_index ON public.agent_chat_messages USING btree (chat_id, message_index);


--
-- Name: idx_agent_chats_owner_canvas_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_chats_owner_canvas_created ON public.agent_chats USING btree (org_id, user_id, canvas_id, created_at DESC);


--
-- Name: agent_chat_messages agent_chat_messages_chat_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_chat_messages
    ADD CONSTRAINT agent_chat_messages_chat_id_fkey FOREIGN KEY (chat_id) REFERENCES public.agent_chats(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

--
-- PostgreSQL database dump
--

\restrict abcdef123

-- Dumped from database version 17.5 (Debian 17.5-1.pgdg130+1)
-- Dumped by pg_dump version 17.9 (Ubuntu 17.9-1.pgdg22.04+1)

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
20260325205949	f
\.


--
-- PostgreSQL database dump complete
--

\unrestrict abcdef123

