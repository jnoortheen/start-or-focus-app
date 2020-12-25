#! /usr/bin/env xonsh

import os
from pathlib import Path

import sh
from loguru import logger

import defopt

current_log = Path(__file__).absolute().parent.joinpath(f"{__name__}.log")
logger.add(current_log, rotation="1 MB",
           retention="2 days")  # Automatically rotate too big file


### todo: activities
# qdbus org.kde.ActivityManager /ActivityManager/Resources/Linking LinkResourceToActivity [agent] [resource] [activity]
# qdbus org.kde.ActivityManager /ActivityManager/Resources/Linking ResourceUnlinkedFromActivity [agent] [resource] [activity]
# qdbus org.kde.KWin /KWin
# https://userbase.kde.org/KDE_System_Administration/PlasmaDesktopScripting
# https://stackoverflow.com/questions/62863205/how-to-list-windows-per-kde-plasma5-activity

def focus_window(program: str):
    return sh.wmctrl("-x", "-a", program)
    # launch krunner in case of multiple windows options
    # qdbus org.kde.krunner /App querySingleRunner windows ""


def search_desktop_file(program: str) -> str:
    import fnmatch
    import re

    reg_expr = re.compile(fnmatch.translate(f"*{program}*.desktop"), re.IGNORECASE)

    data_dirs = os.environ["XDG_DATA_DIRS"].split(os.pathsep)
    for dir in data_dirs:
        path = Path(dir).joinpath("applications")
        if not path.exists():
            continue

        logger.debug(f"searching {path}")

        for root, _, files in os.walk(path, topdown=True):
            for j in files:
                if reg_expr.match(j) and "url-handler" not in j:
                    file_path = os.path.join(root, j)
                    logger.debug(f"found {file_path} matching {reg_expr}")
                    return file_path


def last_line(cmd: sh.RunningCommand) -> str:
    return cmd.stdout.decode().splitlines()[0]


def get_pid_from_pgrep(name: str) -> str:
    cmd = sh.pgrep(name)
    return last_line(cmd)


def xdo_focus_window(class_name: str):
    cmd = sh.xdotool.search("--onlyvisible", "--class", class_name)
    window_id = last_line(cmd)
    sh.xdotool.windowactivate(window_id)


def start_program(program_name: str):
    # other options to open app are dex/xdg-open
    # check if the program is running already
    try:
        focus_window(program_name)
    except Exception as ex:
        logger.info(f"The application {program_name} is not running - {ex}")
        # start the app
        res: sh.RunningCommand = sh.nohup.kioclient5(
            "exec",
            search_desktop_file(program_name),
            _out="/dev/null",
            _err=current_log)
        # subprocess.Popen(
        #     stdout=open('/dev/null', 'w'),
        #     stderr=open(current_log, 'a'),
        #     preexec_fn=os.setpgrp
        # )
        logger.info(f"kioclient ran {res} {res.cmd}")


if __name__ == '__main__':
    defopt.run(start_program)
    # skip garbage collection - https://files.bemusement.org/talks/OSDC2008-FastPython/
    os._exit(0) # it improved around 0.06 seconds

    # it turned out to be very slow
    # we will test it with golang
    # another approach is to use server-client approach
    # https://github.com/janakaud/aws-cli-repl/blob/master/awsr
