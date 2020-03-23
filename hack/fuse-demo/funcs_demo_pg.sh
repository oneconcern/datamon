#! /bin/bash
#
# Some utility functions to run the demo
#

typeset COL_GREEN=$(tput -Txterm setaf 2)
typeset COL_CYAN=$(tput -Txterm setaf 14)
typeset COL_RED=$(tput -Txterm setaf 9)
typeset COL_RESET=$(tput -Txterm sgr0)
typeset dbg=true

dbg_print() {
    if [[ "${dbg}" != "true" ]] ; then
      return
    fi
    echo "${COL_CYAN}DEBUG: $*${COL_RESET}"
}

info_print() {
    echo "${COL_GREEN}INFO: $*${COL_RESET}"
}

error_print() {
    echo "${COL_RED}ERROR: $*${COL_RESET}"
}
