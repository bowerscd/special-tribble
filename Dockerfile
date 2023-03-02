FROM archlinux:latest

RUN pacman-key --init
RUN pacman -Sy archlinux-keyring --noconfirm
RUN pacman -Syu --noconfirm
ADD build/mealbot /usr/bin/mealbot

EXPOSE 80
ENTRYPOINT [ "/usr/bin/mealbot" ]