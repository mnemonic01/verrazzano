TRUNCATE TABLE CONSOLIDATED_SCAN_RESULT;
LOAD DATA LOCAL INFILE '/Users/desagar/Verrazzano/VZ-4761/consolidated.csv'
    INTO TABLE CONSOLIDATED_SCAN_RESULT
    FIELDS
    TERMINATED BY ','
    OPTIONALLY ENCLOSED BY '"'
    LINES
    TERMINATED BY '\n'
    (@commit, @branch, @release, @scantime, @jobnum, @scanner, @vulnid, @sev, @target)
    SET
        VERRAZZANO_COMMIT_HASH = @commit,
        BRANCH_NAME = @branch,
        RELEASE_TAG = @release,
        SCAN_TIME = STR_TO_DATE(@scantime, '%Y%m%d%H%i%s'),
        JOB_NUMBER = @jobnum,
        SCANNER_NAME = @scanner,
        VULNERABILITY_ID = @vulnid,
        SEVERITY = @sev,
        SCAN_TARGET = @target;