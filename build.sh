#! /usr/bin/env bash

BUILD_ROOT=$(cd $(dirname $0); pwd)
OUTPUT_DIR=~/tmp

TEMP_DIR=$(mktemp -d)

log_info() {
    [[ $# -eq 0 ]] && return
    echo $(date +"[%F %T] [INFO]") "$*"
}

log_warn() {
    [[ $# -eq 0 ]] && return
    >&2 echo $(date +"[%F %T] [WARN]") "$*"
}

log_error() {
    [[ $# -eq 0 ]] && return
    >&2 echo $(date +"[%F %T] [ERROR]") "$*"
}

main() {
    local root_dir=$TEMP_DIR/load-agent
    local bin_dir=$root_dir/bin
    local conf_dir=$root_dir/conf
    local logs_dir=$root_dir/logs

    log_info "Building..."

    [[ -d $OUTPUT_DIR ]] && [[ -d $TEMP_DIR ]] && mkdir -p $bin_dir $conf_dir $logs_dir || {
        log_error "Failed to make output directory"
        return 1
    }

    go get -u github.com/geterns/logadpt || {
        log_error "Failed to update dependencies"
        return 1
    }

    cd $BUILD_ROOT/cache && go build -o $bin_dir/cache-agent . || {
        log_error "Failed to build cache-agent"
        return 1
    }

    cd $BUILD_ROOT/load && go build -o $bin_dir/load-agent . || {
        log_error "Failed to build load-agent"
        return 1
    }

    cp $BUILD_ROOT/config/config.json $conf_dir/ || {
        log_error "Failed to copy config file"
        return 1
    }

    cp $BUILD_ROOT/logs/sed.pattern $logs_dir/ || {
        log_error "Failed to copy sed pattern file"
        return 1
    }

    cd $TEMP_DIR && tar -cvzf load-agent.tgz load-agent || {
        log_error "Failed to copy sed pattern file"
        return 1
    }

    mv $TEMP_DIR/load-agent.tgz $OUTPUT_DIR/ || {
        log_error "Failed to move output file"
        return 1
    }

    log_info "Output file:      $OUTPUT_DIR/load-agent.tgz"
    log_info "Fetch command:    wget $(hostname):$OUTPUT_DIR/load-agent.tgz"
    log_info "Release URL:      http://$(hostname):8999/load-agent.tgz"
}

main "$@"
