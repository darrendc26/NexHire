-- NexHire PostgreSQL Database Schema Initialization

-- Enable extension for UUID generation if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    google_id VARCHAR(255) UNIQUE,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    -- picture TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Interviews Table
CREATE TABLE IF NOT EXISTS interviews (
    id VARCHAR(64) PRIMARY KEY,
    recruiter_id VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    description TEXT,
    difficulty VARCHAR(32) NOT NULL DEFAULT 'medium',
    duration INT NOT NULL DEFAULT 30,
    voice_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    share_token VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_recruiter FOREIGN KEY (recruiter_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Candidate Sessions Table
CREATE TABLE IF NOT EXISTS candidate_sessions (
    id VARCHAR(64) PRIMARY KEY,
    interview_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'not_started',
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_interview FOREIGN KEY (interview_id) REFERENCES interviews(id) ON DELETE CASCADE
);


-- Candidate Responses Table (Q&A Turns)
CREATE TABLE IF NOT EXISTS candidate_responses (
    id VARCHAR(64) PRIMARY KEY,
    session_id VARCHAR(64) NOT NULL,
    question TEXT NOT NULL,
    answer TEXT,
    score NUMERIC(4, 2),
    feedback TEXT,
    strengths JSONB,
    weaknesses JSONB,
    question_type VARCHAR(32) DEFAULT 'technical',
    question_order INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_response_session FOREIGN KEY (session_id) REFERENCES candidate_sessions(id) ON DELETE CASCADE
);

-- Candidate Reports Table (Final Evaluation)
CREATE TABLE IF NOT EXISTS candidate_reports (
    id VARCHAR(64) PRIMARY KEY,
    session_id VARCHAR(64) UNIQUE NOT NULL,
    overall_score NUMERIC(5, 2) NOT NULL,
    recommendation VARCHAR(32) NOT NULL,
    summary TEXT NOT NULL,
    strengths JSONB,
    weaknesses JSONB,
    skills JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_report_session FOREIGN KEY (session_id) REFERENCES candidate_sessions(id) ON DELETE CASCADE
);

--Interview Invitation Table
CREATE TABLE IF NOT EXISTS interview_candidates (
    id VARCHAR(64) PRIMARY KEY,
    interview_id VARCHAR(64) NOT NULL,
    email VARCHAR(320) NOT NULL,
    name VARCHAR(255),
    status VARCHAR(32) NOT NULL DEFAULT 'invited',
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_interview_candidate_interview FOREIGN KEY (interview_id) REFERENCES interviews(id) ON DELETE CASCADE,
    CONSTRAINT uq_interview_candidate_email UNIQUE (interview_id, email)
);

CREATE INDEX idx_interview_candidates_email ON interview_candidates(email);



-- Indexes for Query Performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
CREATE INDEX IF NOT EXISTS idx_interviews_recruiter_id ON interviews(recruiter_id);
CREATE INDEX IF NOT EXISTS idx_interviews_share_token ON interviews(share_token);
CREATE INDEX IF NOT EXISTS idx_candidate_sessions_interview_id ON candidate_sessions(interview_id);
CREATE INDEX IF NOT EXISTS idx_candidate_responses_session_id ON candidate_responses(session_id);
CREATE INDEX IF NOT EXISTS idx_candidate_reports_session_id ON candidate_reports(session_id);

