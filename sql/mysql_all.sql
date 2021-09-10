-- phpMyAdmin SQL Dump
-- version 4.9.7
-- https://www.phpmyadmin.net/
--
-- Host: localhost:8787
-- Generation Time: Sep 10, 2021 at 07:24 PM
-- Server version: 5.7.32
-- PHP Version: 7.4.12

SET SQL_MODE = "NO_AUTO_VALUE_ON_ZERO";
SET time_zone = "+00:00";

--
-- Database: `lotteryv2`
--

-- --------------------------------------------------------

--
-- Table structure for table `act`
--

CREATE TABLE `act` (
                       `id` int(11) NOT NULL,
                       `name` varchar(40) NOT NULL,
                       `status` int(1) NOT NULL DEFAULT '0',
                       `open_type` int(1) NOT NULL DEFAULT '1',
                       `open_time` varchar(40) NOT NULL DEFAULT '-1',
                       `end_time` varchar(40) NOT NULL DEFAULT '-1',
                       `admin_by` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `gift`
--

CREATE TABLE `gift` (
                        `id` int(11) NOT NULL,
                        `name` varchar(400) NOT NULL COMMENT '奖品名称',
                        `total` int(5) NOT NULL COMMENT '限定人数',
                        `got` int(5) NOT NULL DEFAULT '0' COMMENT '已获奖人数',
                        `promise` int(5) NOT NULL DEFAULT '0' COMMENT '内定人数',
                        `admin_by` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `lucky_list`
--

CREATE TABLE `lucky_list` (
                              `id` int(11) NOT NULL,
                              `uid` int(1) NOT NULL,
                              `name` varchar(40) NOT NULL,
                              `gift_id` int(5) NOT NULL,
                              `gift_name` varchar(40) NOT NULL,
                              `got_time` varchar(40) NOT NULL,
                              `admin_by` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- --------------------------------------------------------

--
-- Table structure for table `user`
--

CREATE TABLE `user` (
                        `id` int(11) NOT NULL,
                        `name` varchar(40) NOT NULL,
                        `phone_number` varchar(40) NOT NULL,
                        `gift_promise` varchar(5) NOT NULL,
                        `admin_by` int(10) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--
-- Indexes for dumped tables
--

--
-- Indexes for table `act`
--
ALTER TABLE `act`
    ADD PRIMARY KEY (`id`),
  ADD KEY `uid` (`id`);

--
-- Indexes for table `gift`
--
ALTER TABLE `gift`
    ADD PRIMARY KEY (`id`),
  ADD KEY `uid` (`id`);

--
-- Indexes for table `lucky_list`
--
ALTER TABLE `lucky_list`
    ADD PRIMARY KEY (`id`),
  ADD KEY `uid` (`id`);

--
-- Indexes for table `user`
--
ALTER TABLE `user`
    ADD PRIMARY KEY (`id`),
  ADD KEY `uid` (`id`);

--
-- AUTO_INCREMENT for dumped tables
--

--
-- AUTO_INCREMENT for table `act`
--
ALTER TABLE `act`
    MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `gift`
--
ALTER TABLE `gift`
    MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `lucky_list`
--
ALTER TABLE `lucky_list`
    MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `user`
--
ALTER TABLE `user`
    MODIFY `id` int(11) NOT NULL AUTO_INCREMENT;
