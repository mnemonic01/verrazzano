-- Copyright (c) 2022, Oracle and/or its affiliates.
-- Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

-- Each row represents a daily scan job that ran in Jenkins, and that produced results for all releases on a specific branch
CREATE TABLE SCAN_JOB (
                          ID INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
                          JOB_NUMBER SMALLINT,
                          JOB_NAME VARCHAR(30),
                          BRANCH_NAME VARCHAR(30) NOT NULL,
                          UNIQUE (JOB_NUMBER,JOB_NAME,BRANCH_NAME) -- JOB_NUMBER+JOB_NAME+BRANCH_NAME unique key, note that smallint can be up to 65535
);

-- Each row represents a bug in the bug database corresponding to a vulnerability
CREATE TABLE BUG (
                     BUG_ID VARCHAR(12) NOT NULL PRIMARY KEY,
                     BUG_STATUS ENUM('OPEN', 'CLOSED'),
                     VULNERABILITY_ID VARCHAR(30) NOT NULL,
                     RELEASE_TAG VARCHAR(8) NOT NULL
);

-- Each row represents the task of triaging a specific vulnerability
CREATE TABLE TRIAGE_TASK (
                             ID INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
                             VULNERABILITY_ID VARCHAR(30) NOT NULL, -- unique key member this is the vulnerability's identifier e.g. CVE-NNN-NNNN
                             VULNERABILITY_TARGET VARCHAR(100) NOT NULL, -- unique key member
                             RELEASE_TAG VARCHAR(8) NOT NULL, -- unique key member
                             URL VARCHAR(70) NOT NULL,
                             BUG_ID VARCHAR(12), -- FK from BUG table, will be null until BUG_FILED
                             TRIAGE_STATUS ENUM('FALSE_POSITIVE','DOES_NOT_APPLY','NEEDS_TRIAGE','IN_PROGRESS', 'BUG_FILED', 'OTHER_SEE_DETAIL'),
                             TRIAGE_ASSIGNEE VARCHAR(12), -- guid of assignee
                             TRIAGE_DETAIL VARCHAR(50),
                             UNIQUE (VULNERABILITY_ID, VULNERABILITY_TARGET, RELEASE_TAG),
                             FOREIGN KEY (BUG_ID)
                                 REFERENCES BUG(BUG_ID)
                                 ON DELETE CASCADE
);

-- Each row represents one scan result i.e. one vulnerability reported in the results
-- of running a specific scanner in a scan job
CREATE TABLE SCAN_RESULT (
                             ID INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
                             SCAN_JOB_ID INT, -- FK from SCAN_JOB table
                             VERRAZZANO_COMMIT_HASH VARCHAR(40) NOT NULL,
                             RELEASE_TAG VARCHAR(8), -- nullable or "latest" or "periodic" - maybe an enum?
                             SCAN_TIME TIMESTAMP, -- the date/time at which the scan was run
                             TRIAGE_TASK_ID INT, -- FK from TRIAGE_TASK table
                             SCANNER_NAME ENUM('TRIVY', 'GRYPE', 'OCIR', 'OSCS'),
                             SEVERITY ENUM('CRITICAL','HIGH','MEDIUM','LOW','UNKNOWN','NONE','Negligible'),
                             SCAN_TARGET VARCHAR(100) NOT NULL, -- the Docker image name and tag or other scan target (source code? etc)
                             SCAN_TARGET_COMMIT_HASH VARCHAR(40) NOT NULL,
                             FOREIGN KEY (SCAN_JOB_ID)
                                 REFERENCES SCAN_JOB(ID)
                                 ON DELETE CASCADE,
                             FOREIGN KEY (TRIAGE_TASK_ID)
                                 REFERENCES TRIAGE_TASK(ID)
                                 ON DELETE CASCADE
);

-- Each row represents a Vulnerability/Target/Release combination that is either a known false positive
-- or is suppressed for some other reason
CREATE TABLE SUPPRESSION (
                             ID INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
                             VULNERABILITY_ID VARCHAR(30) NOT NULL, -- unique key member
                             TARGET VARCHAR(100) NOT NULL, -- unique key member
                             RELEASE_TAG VARCHAR(8) NOT NULL, -- unique key member
                             REASON ENUM('FALSE_POSITIVE','WILL_NOT_FIX') NOT NULL,
                             UNIQUE(VULNERABILITY_ID, TARGET, RELEASE_TAG)
);

-- This table is where data is initially directly loaded from the CSV for further processing to split it out to
-- other tables. After loading and processing the data, the table will be truncated (for the next batch of data to come in).
-- Each row represents a row in the CSV file
CREATE TABLE CONSOLIDATED_SCAN_RESULT (
                                          ID INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
                                          VERRAZZANO_COMMIT_HASH VARCHAR(40) NOT NULL,
                                          BRANCH_NAME VARCHAR(30) NOT NULL,
                                          RELEASE_TAG VARCHAR(8) NOT NULL, -- values: latest, periodic or feature, or a release tag
                                          SCAN_TIME TIMESTAMP NOT NULL,
                                          JOB_NUMBER SMALLINT NOT NULL,
                                          SCANNER_NAME ENUM('TRIVY', 'GRYPE', 'OCIR', 'OSCS') NOT NULL,
                                          VULNERABILITY_ID VARCHAR(30) NOT NULL,
                                          SEVERITY ENUM('CRITICAL','HIGH','MEDIUM','LOW','UNKNOWN', 'NONE', 'Negligible') NOT NULL,
                                          SCAN_TARGET VARCHAR(100)
);
