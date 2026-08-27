package fbproto

const (
	ISC_TIME_SECONDS_PRECISION = 10000

	// Protocol Types (accept_type)
	Ptype_batch_send    = 3 // Batch sends, no asynchrony
	Ptype_out_of_band   = 4 // Batch sends w/ out of band notification
	Ptype_lazy_send     = 5 // Deferred packets delivery
	Ptype_MASK          = 0xFF
	Pflag_compress      = 0x100
	Pflag_win_sspi_nego = 0x200

	P_cnct_min_type = 0

	GenericArchitectureClient = 1

	ConnectVersion3 = 3

	// Protocol Version
	PROTOCOL_VERSION10 = 10
	PROTOCOL_VERSION11 = 11
	PROTOCOL_VERSION12 = 12
	PROTOCOL_VERSION13 = 13
	PROTOCOL_VERSION14 = 14
	PROTOCOL_VERSION15 = 15
	PROTOCOL_VERSION16 = 16
	PROTOCOL_VERSION17 = 17
	PROTOCOL_VERSION18 = 18
	PROTOCOL_VERSION19 = 19

	ProtocolFlag = 0x8000
	ProtocolMask = ^ProtocolFlag & 0xFFFF

	CNCT_user              = 1
	CNCT_passwd            = 2
	CNCT_host              = 4
	CNCT_group             = 5
	CNCT_user_verification = 6
	CNCT_specific_data     = 7
	CNCT_plugin_name       = 8
	CNCT_login             = 9
	CNCT_plugin_list       = 10
	CNCT_client_crypt      = 11

	Isc_info_end            = 1
	Isc_info_truncated      = 2
	Isc_info_error          = 3
	Isc_info_data_not_ready = 4
	Isc_info_length         = 126
	Isc_info_flag_end       = 127

	Isc_info_blob_num_segments = 4
	Isc_info_blob_max_segment  = 5
	Isc_info_blob_total_length = 6
	Isc_info_blob_type         = 7

	Isc_info_db_id                 = 4
	Isc_info_reads                 = 5
	Isc_info_writes                = 6
	Isc_info_fetches               = 7
	Isc_info_marks                 = 8
	Isc_info_implementation        = 11
	Isc_info_version               = 12
	Isc_info_base_level            = 13
	Isc_info_page_size             = 14
	Isc_info_num_buffers           = 15
	Isc_info_limbo                 = 16
	Isc_info_current_memory        = 17
	Isc_info_max_memory            = 18
	Isc_info_window_turns          = 19
	Isc_info_license               = 20
	Isc_info_allocation            = 21
	Isc_info_attachment_id         = 22
	Isc_info_read_seq_count        = 23
	Isc_info_read_idx_count        = 24
	Isc_info_insert_count          = 25
	Isc_info_update_count          = 26
	Isc_info_delete_count          = 27
	Isc_info_backout_count         = 28
	Isc_info_purge_count           = 29
	Isc_info_expunge_count         = 30
	Isc_info_sweep_interval        = 31
	Isc_info_ods_version           = 32
	Isc_info_ods_minor_version     = 33
	Isc_info_no_reserve            = 34
	Isc_info_logfile               = 35
	Isc_info_cur_logfile_name      = 36
	Isc_info_cur_log_part_offset   = 37
	Isc_info_num_wal_buffers       = 38
	Isc_info_wal_buffer_size       = 39
	Isc_info_wal_ckpt_length       = 40
	Isc_info_wal_cur_ckpt_interval = 41
	Isc_info_wal_prv_ckpt_fname    = 42
	Isc_info_wal_prv_ckpt_poffset  = 43
	Isc_info_wal_recv_ckpt_fname   = 44
	Isc_info_wal_recv_ckpt_poffset = 45
	Isc_info_wal_grpc_wait_usecs   = 47
	Isc_info_wal_num_io            = 48
	Isc_info_wal_avg_io_size       = 49
	Isc_info_wal_num_commits       = 50
	Isc_info_wal_avg_grpc_size     = 51
	Isc_info_forced_writes         = 52
	Isc_info_user_names            = 53
	Isc_info_page_errors           = 54
	Isc_info_record_errors         = 55
	Isc_info_bpage_errors          = 56
	Isc_info_dpage_errors          = 57
	Isc_info_ipage_errors          = 58
	Isc_info_ppage_errors          = 59
	Isc_info_tpage_errors          = 60
	Isc_info_set_page_buffers      = 61
	Isc_info_db_sql_dialect        = 62
	Isc_info_db_read_only          = 63
	Isc_info_db_size_in_pages      = 64
	Isc_info_att_charset           = 101
	Isc_info_db_class              = 102
	Isc_info_firebird_version      = 103
	Isc_info_oldest_transaction    = 104
	Isc_info_oldest_active         = 105
	Isc_info_oldest_snapshot       = 106
	Isc_info_next_transaction      = 107
	Isc_info_db_provider           = 108
	Isc_info_active_transactions   = 109
	Isc_info_active_tran_count     = 110
	Isc_info_creation_date         = 111
	Isc_info_db_file_size          = 112

	// isc_info_sql_records items
	Isc_info_req_select_count = 13
	Isc_info_req_insert_count = 14
	Isc_info_req_update_count = 15
	Isc_info_req_delete_count = 16

	Isc_info_svc_svr_db_info        = 50
	Isc_info_svc_get_license        = 51
	Isc_info_svc_get_license_mask   = 52
	Isc_info_svc_get_config         = 53
	Isc_info_svc_version            = 54
	Isc_info_svc_server_version     = 55
	Isc_info_svc_implementation     = 56
	Isc_info_svc_capabilities       = 57
	Isc_info_svc_user_dbpath        = 58
	Isc_info_svc_get_env            = 59
	Isc_info_svc_get_env_lock       = 60
	Isc_info_svc_get_env_msg        = 61
	Isc_info_svc_line               = 62
	Isc_info_svc_to_eof             = 63
	Isc_info_svc_timeout            = 64
	Isc_info_svc_get_licensed_users = 65
	Isc_info_svc_limbo_trans        = 66
	Isc_info_svc_running            = 67
	Isc_info_svc_get_users          = 68

	Isc_tpb_version1         = 1
	Isc_tpb_version3         = 3
	Isc_tpb_consistency      = 1
	Isc_tpb_concurrency      = 2
	Isc_tpb_shared           = 3
	Isc_tpb_protected        = 4
	Isc_tpb_exclusive        = 5
	Isc_tpb_wait             = 6
	Isc_tpb_nowait           = 7
	Isc_tpb_read             = 8
	Isc_tpb_write            = 9
	Isc_tpb_lock_read        = 10
	Isc_tpb_lock_write       = 11
	Isc_tpb_verb_time        = 12
	Isc_tpb_commit_time      = 13
	Isc_tpb_ignore_limbo     = 14
	Isc_tpb_read_committed   = 15
	Isc_tpb_autocommit       = 16
	Isc_tpb_rec_version      = 17
	Isc_tpb_no_rec_version   = 18
	Isc_tpb_restart_requests = 19
	Isc_tpb_no_auto_undo     = 20
	Isc_tpb_lock_timeout     = 21

	// Service Parameter Block parameter
	Isc_spb_version1        = 1
	Isc_spb_current_version = 2
	Isc_spb_version         = Isc_spb_current_version

	// Isc_spb_version3 identifies a wide service attachment parameter buffer.
	Isc_spb_version3 = 3

	Isc_spb_user_name         = 28 // Isc_dpb_user_name
	Isc_spb_sys_user_name     = 19 // isc_dpb_sys_user_name
	Isc_spb_sys_user_name_enc = 31 // isc_dpb_sys_user_name_enc
	Isc_spb_password          = 29 // Isc_dpb_password
	Isc_spb_password_enc      = 30 // isc_dpb_password_enc
	Isc_spb_command_line      = 105
	Isc_spb_dbname            = 106
	Isc_spb_verbose           = 107
	Isc_spb_options           = 108
	Isc_spb_address_path      = 109
	Isc_spb_process_id        = 110
	Isc_spb_trusted_auth      = 111

	// Isc_spb_specific_auth_data carries plugin-specific authentication data.
	Isc_spb_specific_auth_data = Isc_spb_trusted_auth

	Isc_spb_process_name     = 112
	Isc_spb_trusted_role     = 113
	Isc_spb_verbint          = 114
	Isc_spb_auth_block       = 115
	Isc_spb_auth_plugin_name = 116
	Isc_spb_auth_plugin_list = 117
	Isc_spb_utf8_filename    = 118
	Isc_spb_client_version   = 119
	Isc_spb_remote_protocol  = 120
	Isc_spb_host_name        = 121
	Isc_spb_os_user          = 122
	Isc_spb_config           = 123
	Isc_spb_expected_db      = 124

	Isc_spb_connect_timeout       = 57 // isc_dpb_connect_timeout
	Isc_spb_dummy_packet_interval = 58 // isc_dpb_dummy_packet_interval
	Isc_spb_sql_role_name         = 60 // Isc_dpb_sql_role_name

	// Parameters for isc_action_{add|del|mod|disp)_user
	Isc_spb_sec_userid     = 5
	Isc_spb_sec_groupid    = 6
	Isc_spb_sec_username   = 7
	Isc_spb_sec_password   = 8
	Isc_spb_sec_groupname  = 9
	Isc_spb_sec_firstname  = 10
	Isc_spb_sec_middlename = 11
	Isc_spb_sec_lastname   = 12
	Isc_spb_sec_admin      = 13

	//Database Parameter Block Types
	Isc_dpb_version1              = 1
	Isc_dpb_version2              = 2
	Isc_dpb_page_size             = 4
	Isc_dpb_num_buffers           = 5
	Isc_dpb_force_write           = 24
	Isc_dpb_user_name             = 28
	Isc_dpb_password              = 29
	Isc_dpb_password_enc          = 30
	Isc_dpb_lc_ctype              = 48
	Isc_dpb_overwrite             = 54
	Isc_dpb_connect_timeout       = 57
	Isc_dpb_dummy_packet_interval = 58
	Isc_dpb_sql_role_name         = 60
	Isc_dpb_set_page_buffers      = 61
	Isc_dpb_sql_dialect           = 63
	Isc_dpb_set_db_sql_dialect    = 65
	Isc_dpb_set_db_charset        = 68
	Isc_dpb_process_id            = 71
	Isc_dpb_no_db_triggers        = 72
	Isc_dpb_trusted_auth          = 73
	Isc_dpb_process_name          = 74
	Isc_dpb_utf8_filename         = 77
	Isc_dpb_client_version        = 80
	Isc_dpb_host_name             = 82
	Isc_dpb_os_user               = 83
	Isc_dpb_specific_auth_data    = 84
	Isc_dpb_auth_plugin_list      = 85
	Isc_dpb_auth_plugin_name      = 86
	Isc_dpb_config                = 87
	Isc_dpb_nolinger              = 88
	Isc_dpb_reset_icu             = 89
	Isc_dpb_map_attach            = 90
	Isc_dpb_session_time_zone     = 91
	Isc_dpb_max_blob_cache_size   = 103
	Isc_dpb_max_inline_blob_size  = 104
	Isc_dpb_search_path           = 105

	// backup
	Isc_spb_bkp_file                = 5
	Isc_spb_bkp_factor              = 6
	Isc_spb_bkp_length              = 7
	Isc_spb_bkp_skip_data           = 8
	Isc_spb_bkp_stat                = 15
	Isc_spb_bkp_keyholder           = 16
	Isc_spb_bkp_keyname             = 17
	Isc_spb_bkp_crypt               = 18
	Isc_spb_bkp_include_data        = 19
	Isc_spb_bkp_parallel_workers    = 21
	Isc_spb_bkp_skip_schema_data    = 22
	Isc_spb_bkp_include_schema_data = 23
	Isc_spb_bkp_ignore_checksums    = 0x01
	Isc_spb_bkp_ignore_limbo        = 0x02
	Isc_spb_bkp_metadata_only       = 0x04
	Isc_spb_bkp_no_garbage_collect  = 0x08
	Isc_spb_bkp_old_descriptions    = 0x10
	Isc_spb_bkp_non_transportable   = 0x20
	Isc_spb_bkp_convert             = 0x40
	Isc_spb_bkp_expand              = 0x80
	Isc_spb_bkp_no_triggers         = 0x8000
	Isc_spb_bkp_zip                 = 0x010000
	Isc_spb_bkp_direct_io           = 0x020000
	Isc_spb_bkp_compression         = 102
	Isc_spb_bkp_replace             = 103

	// restore
	Isc_spb_res_buffers             = 9
	Isc_spb_res_page_size           = 10
	Isc_spb_res_length              = 11
	Isc_spb_res_access_mode         = 12
	Isc_spb_res_fix_fss_data        = 13
	Isc_spb_res_fix_fss_metadata    = 14
	Isc_spb_res_skip_data           = Isc_spb_bkp_skip_data
	Isc_spb_res_include_data        = Isc_spb_bkp_include_data
	Isc_spb_res_skip_schema_data    = Isc_spb_bkp_skip_schema_data
	Isc_spb_res_include_schema_data = Isc_spb_bkp_include_schema_data
	Isc_spb_res_keyholder           = Isc_spb_bkp_keyholder
	Isc_spb_res_keyname             = Isc_spb_bkp_keyname
	Isc_spb_res_crypt               = Isc_spb_bkp_crypt
	Isc_spb_res_stat                = Isc_spb_bkp_stat
	Isc_spb_res_replica_mode        = 20
	Isc_spb_res_parallel_workers    = Isc_spb_bkp_parallel_workers
	Isc_spb_res_metadata_only       = Isc_spb_bkp_metadata_only
	Isc_spb_res_deactivate_idx      = 0x0100
	Isc_spb_res_no_shadow           = 0x0200
	Isc_spb_res_no_validity         = 0x0400
	Isc_spb_res_one_at_a_time       = 0x0800
	Isc_spb_res_replace             = 0x1000
	Isc_spb_res_create              = 0x2000
	Isc_spb_res_use_all_space       = 0x4000
	Isc_spb_res_direct_io           = Isc_spb_bkp_direct_io
	Isc_spb_res_ts_original_paths   = 0x800000
	Isc_spb_res_am_readonly         = Isc_spb_prp_am_readonly
	Isc_spb_res_am_readwrite        = Isc_spb_prp_am_readwrite
	Isc_spb_res_rm_none             = Isc_spb_prp_rm_none
	Isc_spb_res_rm_readonly         = Isc_spb_prp_rm_readonly
	Isc_spb_res_rm_readwrite        = Isc_spb_prp_rm_readwrite
	Isc_spb_res_ts_mapping_file     = 120
	Isc_spb_res_ts_path             = 121

	// Parameters for isc_action_svc_nbak
	Isc_spb_nbk_level         = 5
	Isc_spb_nbk_file          = 6
	Isc_spb_nbk_direct        = 7
	Isc_spb_nbk_guid          = 8
	Isc_spb_nbk_clean_history = 9
	Isc_spb_nbk_keep_days     = 10
	Isc_spb_nbk_keep_rows     = 11
	Isc_spb_nbk_no_triggers   = 0x01
	Isc_spb_nbk_inplace       = 0x02
	Isc_spb_nbk_sequence      = 0x04

	// Parameters for isc_action_svc_properties
	Isc_spb_prp_page_buffers          = 5
	Isc_spb_prp_sweep_interval        = 6
	Isc_spb_prp_shutdown_db           = 7
	Isc_spb_prp_deny_new_attachments  = 9
	Isc_spb_prp_deny_new_transactions = 10
	Isc_spb_prp_reserve_space         = 11
	Isc_spb_prp_write_mode            = 12
	Isc_spb_prp_access_mode           = 13
	Isc_spb_prp_set_sql_dialect       = 14
	Isc_spb_prp_activate              = 0x0100
	Isc_spb_prp_db_online             = 0x0200
	Isc_spb_prp_nolinger              = 0x0400
	Isc_spb_prp_force_shutdown        = 41
	Isc_spb_prp_attachments_shutdown  = 42
	Isc_spb_prp_transactions_shutdown = 43
	Isc_spb_prp_shutdown_mode         = 44
	Isc_spb_prp_online_mode           = 45
	Isc_spb_prp_replica_mode          = 46

	// Parameters for isc_spb_prp_shutdown_mode and isc_spb_prp_online_mode
	Isc_spb_prp_sm_normal = 0
	Isc_spb_prp_sm_multi  = 1
	Isc_spb_prp_sm_single = 2
	Isc_spb_prp_sm_full   = 3

	// Parameters for isc_spb_prp_replica_mode
	Isc_spb_prp_rm_none      = 0
	Isc_spb_prp_rm_readonly  = 1
	Isc_spb_prp_rm_readwrite = 2

	// Parameters for isc_action_svc_repair
	Isc_spb_rpr_commit_trans         = 15
	Isc_spb_rpr_rollback_trans       = 34
	Isc_spb_rpr_recover_two_phase    = 17
	Isc_spb_tra_id                   = 18
	Isc_spb_single_tra_id            = 19
	Isc_spb_multi_tra_id             = 20
	Isc_spb_tra_state                = 21
	Isc_spb_tra_state_limbo          = 22
	Isc_spb_tra_state_commit         = 23
	Isc_spb_tra_state_rollback       = 24
	Isc_spb_tra_state_unknown        = 25
	Isc_spb_tra_host_site            = 26
	Isc_spb_tra_remote_site          = 27
	Isc_spb_tra_db_path              = 28
	Isc_spb_tra_advise               = 29
	Isc_spb_tra_advise_commit        = 30
	Isc_spb_tra_advise_rollback      = 31
	Isc_spb_tra_advise_unknown       = 33
	Isc_spb_tra_id_64                = 46
	Isc_spb_single_tra_id_64         = 47
	Isc_spb_multi_tra_id_64          = 48
	Isc_spb_rpr_commit_trans_64      = 49
	Isc_spb_rpr_rollback_trans_64    = 50
	Isc_spb_rpr_recover_two_phase_64 = 51
	Isc_spb_rpr_par_workers          = 52

	// Parameters for isc_spb_prp_reserve_space *
	Isc_spb_prp_res_use_full = 35
	Isc_spb_prp_res          = 36

	// Parameters for isc_spb_prp_write_mode
	Isc_spb_prp_wm_async = 37
	Isc_spb_prp_wm_sync  = 38

	// Parameters for isc_spb_prp_access_mode
	Isc_spb_prp_am_readonly  = 39
	Isc_spb_prp_am_readwrite = 40

	// trace
	Isc_spb_trc_id   = 1
	Isc_spb_trc_name = 2
	Isc_spb_trc_cfg  = 3

	// isc_info_svc_svr_db_info params
	Isc_spb_num_att = 5
	Isc_spb_num_db  = 6

	// isc_info_svc_db_stats params
	Isc_spb_sts_data_pages      = 0x01
	Isc_spb_sts_db_log          = 0x02
	Isc_spb_sts_hdr_pages       = 0x04
	Isc_spb_sts_idx_pages       = 0x08
	Isc_spb_sts_sys_relations   = 0x10
	Isc_spb_sts_record_versions = 0x20
	Isc_spb_sts_table           = 0x40
	Isc_spb_sts_nocreation      = 0x80

	// isc_action_svc_repair params
	Isc_spb_rpr_validate_db      = 0x01
	Isc_spb_rpr_sweep_db         = 0x02
	Isc_spb_rpr_mend_db          = 0x04
	Isc_spb_rpr_list_limbo_trans = 0x08
	Isc_spb_rpr_check_db         = 0x10
	Isc_spb_rpr_ignore_checksum  = 0x20
	Isc_spb_rpr_kill_shadows     = 0x40
	Isc_spb_rpr_full             = 0x80

	// BLR params
	Isc_blr_short           = 7
	Isc_blr_long            = 8
	Isc_blr_quad            = 9
	Isc_blr_float           = 10
	Isc_blr_d_float         = 11
	Isc_blr_sql_date        = 12
	Isc_blr_sql_time        = 13
	Isc_blr_text            = 14
	Isc_blr_text2           = 15
	Isc_blr_int64           = 16
	Isc_blr_bool            = 23
	Isc_blr_dec64           = 24
	Isc_blr_dec128          = 25
	Isc_blr_int128          = 26
	Isc_blr_double          = 27
	Isc_blr_sql_time_tz     = 28
	Isc_blr_timestamp_tz    = 29
	Isc_blr_ex_time_tz      = 30
	Isc_blr_ex_timestamp_tz = 31
	Isc_blr_timestamp       = 35
	Isc_blr_varying         = 37
	Isc_blr_varying2        = 38
	Isc_blr_cstring         = 40
	Isc_blr_cstring2        = 41
	Isc_blr_blob            = 261

	Isc_blr_version4 = 4
	Isc_blr_version5 = 5

	Isc_blr_eoc = 76
	Isc_blr_end = 255

	Isc_blr_assignment   = 1
	Isc_blr_begin        = 2
	Isc_blr_dcl_variable = 3
	Isc_blr_message      = 4
	Isc_blr_erase        = 5
	Isc_blr_for          = 7
	Isc_blr_if           = 8
	Isc_blr_loop         = 9
	Isc_blr_modify       = 10
	Isc_blr_handler      = 11
	Isc_blr_receive      = 12
	Isc_blr_select       = 13
	Isc_blr_send         = 14
	Isc_blr_store        = 15
	Isc_blr_label        = 17
	Isc_blr_leave        = 18
	Isc_blr_store2       = 19
	Isc_blr_post         = 20
	Isc_blr_literal      = 21
	Isc_blr_dbkey        = 22
	Isc_blr_field        = 23
	Isc_blr_fid          = 24
	Isc_blr_parameter    = 25
	Isc_blr_variable     = 26
	Isc_blr_average      = 27
	Isc_blr_count        = 28
	Isc_blr_maximum      = 29
	Isc_blr_minimum      = 30
	Isc_blr_total        = 31

	Isc_blr_add         = 34
	Isc_blr_subtract    = 35
	Isc_blr_multiply    = 36
	Isc_blr_divide      = 37
	Isc_blr_negate      = 38
	Isc_blr_concatenate = 39
	Isc_blr_substring   = 40
	Isc_blr_parameter2  = 41
	Isc_blr_from        = 42
	Isc_blr_via         = 43
	Isc_blr_user_name   = 44
	Isc_blr_null        = 45
	Isc_blr_equiv       = 46
	Isc_blr_eql         = 47
	Isc_blr_neq         = 48
	Isc_blr_gtr         = 49
	Isc_blr_geq         = 50
	Isc_blr_lss         = 51
	Isc_blr_leq         = 52
	Isc_blr_containing  = 53
	Isc_blr_matching    = 54
	Isc_blr_starting    = 55
	Isc_blr_between     = 56
	Isc_blr_or          = 57
	Isc_blr_and         = 58
	Isc_blr_not         = 59
	Isc_blr_any         = 60
	Isc_blr_missing     = 61
	Isc_blr_unique      = 62
	Isc_blr_like        = 63

	Isc_blr_rse        = 67
	Isc_blr_first      = 68
	Isc_blr_project    = 69
	Isc_blr_sort       = 70
	Isc_blr_boolean    = 71
	Isc_blr_ascending  = 72
	Isc_blr_descending = 73
	Isc_blr_relation   = 74
	Isc_blr_rid        = 75
	Isc_blr_union      = 76
	Isc_blr_map        = 77
	Isc_blr_group_by   = 78
	Isc_blr_aggregate  = 79
	Isc_blr_join_type  = 80

	Isc_blr_agg_count   = 83
	Isc_blr_agg_max     = 84
	Isc_blr_agg_min     = 85
	Isc_blr_agg_total   = 86
	Isc_blr_agg_average = 87

	Isc_blr_agg_count2           = 93
	Isc_blr_agg_count_distinct   = 94
	Isc_blr_agg_total_distinct   = 95
	Isc_blr_agg_average_distinct = 96

	Isc_blr_function   = 100
	Isc_blr_gen_id     = 101
	Isc_blr_upcase     = 103
	Isc_blr_value_if   = 105
	Isc_blr_matching2  = 106
	Isc_blr_index      = 107
	Isc_blr_ansi_like  = 108
	Isc_blr_scrollable = 109

	Isc_blr_rs_stream = 119
	Isc_blr_exec_proc = 120

	Isc_blr_procedure       = 124
	Isc_blr_pid             = 125
	Isc_blr_exec_pid        = 126
	Isc_blr_singular        = 127
	Isc_blr_abort           = 128
	Isc_blr_block           = 129
	Isc_blr_error_handler   = 130
	Isc_blr_cast            = 131
	Isc_blr_pid2            = 132
	Isc_blr_procedure2      = 133
	Isc_blr_start_savepoint = 134
	Isc_blr_end_savepoint   = 135

	Isc_blr_plan         = 139
	Isc_blr_merge        = 140
	Isc_blr_join         = 141
	Isc_blr_sequential   = 142
	Isc_blr_navigational = 143
	Isc_blr_indices      = 144
	Isc_blr_retrieve     = 145
	Isc_blr_relation2    = 146
	Isc_blr_rid2         = 147

	Isc_blr_set_generator = 150
	Isc_blr_ansi_any      = 151
	Isc_blr_exists        = 152

	Isc_blr_record_version = 154
	Isc_blr_stall          = 155

	Isc_blr_ansi_all = 158
	Isc_blr_extract  = 159

	Isc_blr_extract_year        = 0
	Isc_blr_extract_month       = 1
	Isc_blr_extract_day         = 2
	Isc_blr_extract_hour        = 3
	Isc_blr_extract_minute      = 4
	Isc_blr_extract_second      = 5
	Isc_blr_extract_weekday     = 6
	Isc_blr_extract_yearday     = 7
	Isc_blr_extract_millisecond = 8
	Isc_blr_extract_week        = 9
	Isc_blr_current_date        = 160
	Isc_blr_current_timestamp   = 161
	Isc_blr_current_time        = 162

	Isc_blr_post_arg           = 163
	Isc_blr_exec_into          = 164
	Isc_blr_dcl_cursor         = 166
	Isc_blr_cursor_stmt        = 167
	Isc_blr_current_timestamp2 = 168
	Isc_blr_current_time2      = 169
	Isc_blr_agg_list           = 170
	Isc_blr_agg_list_distinct  = 171
	Isc_blr_modify2            = 172

	Isc_blr_current_role = 174
	Isc_blr_skip         = 175

	Isc_blr_exec_sql      = 176
	Isc_blr_internal_info = 177
	Isc_blr_nullsfirst    = 178
	Isc_blr_writelock     = 179
	Isc_blr_nullslast     = 180

	Isc_blr_lowcase = 181
	Isc_blr_strlen  = 182

	Isc_blr_strlen_bit   = 0
	Isc_blr_strlen_char  = 1
	Isc_blr_strlen_octet = 2
	Isc_blr_trim         = 183

	Isc_blr_trim_both     = 0
	Isc_blr_trim_leading  = 1
	Isc_blr_trim_trailing = 2

	Isc_blr_trim_spaces     = 0
	Isc_blr_trim_characters = 1

	Isc_blr_cursor_open         = 0
	Isc_blr_cursor_close        = 1
	Isc_blr_cursor_fetch        = 2
	Isc_blr_cursor_fetch_scroll = 3

	Isc_blr_scroll_forward  = 0
	Isc_blr_scroll_backward = 1
	Isc_blr_scroll_bof      = 2
	Isc_blr_scroll_eof      = 3
	Isc_blr_scroll_absolute = 4
	Isc_blr_scroll_relative = 5

	Isc_blr_init_variable = 184
	Isc_blr_recurse       = 185
	Isc_blr_sys_function  = 186

	Isc_blr_auto_trans = 187
	Isc_blr_similar    = 188
	Isc_blr_exec_stmt  = 189

	Isc_blr_exec_stmt_inputs     = 1
	Isc_blr_exec_stmt_outputs    = 2
	Isc_blr_exec_stmt_sql        = 3
	Isc_blr_exec_stmt_proc_block = 4
	Isc_blr_exec_stmt_data_src   = 5
	Isc_blr_exec_stmt_user       = 6
	Isc_blr_exec_stmt_pwd        = 7
	Isc_blr_exec_stmt_tran       = 8
	Isc_blr_exec_stmt_tran_clone = 9
	Isc_blr_exec_stmt_privs      = 10
	Isc_blr_exec_stmt_in_params  = 11
	Isc_blr_exec_stmt_in_params2 = 12
	Isc_blr_exec_stmt_out_params = 13
	Isc_blr_exec_stmt_role       = 14
	Isc_blr_stmt_expr            = 190
	Isc_blr_derived_expr         = 191

	Isc_blr_procedure3        = 192
	Isc_blr_exec_proc2        = 193
	Isc_blr_function2         = 194
	Isc_blr_window            = 195
	Isc_blr_partition_by      = 196
	Isc_blr_continue_loop     = 197
	Isc_blr_procedure4        = 198
	Isc_blr_agg_function      = 199
	Isc_blr_substring_similar = 200
	Isc_blr_bool_as_value     = 201
	Isc_blr_coalesce          = 202
	Isc_blr_decode            = 203
	Isc_blr_exec_subproc      = 204
	Isc_blr_subproc_decl      = 205
	Isc_blr_subproc           = 206
	Isc_blr_subfunc_decl      = 207
	Isc_blr_subfunc           = 208
	Isc_blr_record_version2   = 209

	// SDL params
	Isc_sdl_version1      = 1
	Isc_sdl_relation      = 2
	Isc_sdl_field         = 4
	Isc_sdl_struct        = 6
	Isc_sdl_variable      = 7
	Isc_sdl_scalar        = 8
	Isc_sdl_tiny_integer  = 9
	Isc_sdl_long_integer  = 11
	Isc_sdl_do2           = 34
	Isc_sdl_do1           = 35
	Isc_sdl_schema        = 37
	Isc_sdl_short_integer = 10
	Isc_sdl_element       = 36
	Isc_sdl_eoc           = 255

	// Service Action Items
	Isc_action_svc_backup           = 1
	Isc_action_svc_restore          = 2
	Isc_action_svc_repair           = 3
	Isc_action_svc_add_user         = 4
	Isc_action_svc_delete_user      = 5
	Isc_action_svc_modify_user      = 6
	Isc_action_svc_display_user     = 7
	Isc_action_svc_properties       = 8
	Isc_action_svc_add_license      = 9
	Isc_action_svc_remove_license   = 10
	Isc_action_svc_db_stats         = 11
	Isc_action_svc_get_ib_log       = 12
	Isc_action_svc_get_fb_log       = 12
	Isc_action_svc_nbak             = 20
	Isc_action_svc_nrest            = 21
	Isc_action_svc_trace_start      = 22
	Isc_action_svc_trace_stop       = 23
	Isc_action_svc_trace_suspend    = 24
	Isc_action_svc_trace_resume     = 25
	Isc_action_svc_trace_list       = 26
	Isc_action_svc_set_mapping      = 27
	Isc_action_svc_drop_mapping     = 28
	Isc_action_svc_display_user_adm = 29
	Isc_action_svc_validate         = 30
	Isc_action_svc_nfix             = 31
	Isc_action_svc_last             = 32

	// Transaction information items
	Isc_info_tra_id                 = 4
	Isc_info_tra_oldest_interesting = 5
	Isc_info_tra_oldest_snapshot    = 6
	Isc_info_tra_oldest_active      = 7
	Isc_info_tra_isolation          = 8
	Isc_info_tra_access             = 9
	Isc_info_tra_lock_timeout       = 10

	// SQL information items
	Isc_info_sql_select          = 4
	Isc_info_sql_bind            = 5
	Isc_info_sql_num_variables   = 6
	Isc_info_sql_describe_vars   = 7
	Isc_info_sql_describe_end    = 8
	Isc_info_sql_sqlda_seq       = 9
	Isc_info_sql_message_seq     = 10
	Isc_info_sql_type            = 11
	Isc_info_sql_sub_type        = 12
	Isc_info_sql_scale           = 13
	Isc_info_sql_length          = 14
	Isc_info_sql_null_ind        = 15
	Isc_info_sql_field           = 16
	Isc_info_sql_relation        = 17
	Isc_info_sql_owner           = 18
	Isc_info_sql_alias           = 19
	Isc_info_sql_sqlda_start     = 20
	Isc_info_sql_stmt_type       = 21
	Isc_info_sql_get_plan        = 22
	Isc_info_sql_records         = 23
	Isc_info_sql_batch_fetch     = 24
	Isc_info_sql_relation_alias  = 25
	Isc_info_sql_relation_schema = 33

	Isc_info_sql_stmt_select         = 1
	Isc_info_sql_stmt_insert         = 2
	Isc_info_sql_stmt_update         = 3
	Isc_info_sql_stmt_delete         = 4
	Isc_info_sql_stmt_ddl            = 5
	Isc_info_sql_stmt_get_segment    = 6
	Isc_info_sql_stmt_put_segment    = 7
	Isc_info_sql_stmt_exec_procedure = 8
	Isc_info_sql_stmt_start_trans    = 9
	Isc_info_sql_stmt_commit         = 10
	Isc_info_sql_stmt_rollback       = 11
	Isc_info_sql_stmt_select_for_upd = 12
	Isc_info_sql_stmt_set_generator  = 13
	Isc_info_sql_stmt_savepoint      = 14

	Isc_arg_end         = 0
	Isc_arg_gds         = 1
	Isc_arg_string      = 2
	Isc_arg_cstring     = 3
	Isc_arg_number      = 4
	Isc_arg_interpreted = 5
	Isc_arg_vms         = 6
	Isc_arg_unix        = 7
	Isc_arg_domain      = 8
	Isc_arg_dos         = 9
	Isc_arg_mpexl       = 10
	Isc_arg_mpexl_ipc   = 11
	Isc_arg_next_mach   = 15
	Isc_arg_netware     = 16
	Isc_arg_win32       = 17
	Isc_arg_warning     = 18
	Isc_arg_sql_state   = 19

	Isc_bpb_version1      = 1
	Isc_bpb_source_type   = 1
	Isc_bpb_target_type   = 2
	Isc_bpb_type          = 3
	Isc_bpb_source_interp = 4
	Isc_bpb_target_interp = 5
	Isc_bpb_filter_param  = 6

	Isc_bpb_type_segmented = 0
	Isc_bpb_type_stream    = 1
	Isc_bpb_storage_temp   = 2

	Op_connect            = 1
	Op_exit               = 2
	Op_accept             = 3
	Op_reject             = 4
	Op_protocol           = 5
	Op_disconnect         = 6
	Op_response           = 9
	Op_attach             = 19
	Op_create             = 20
	Op_detach             = 21
	Op_transaction        = 29
	Op_commit             = 30
	Op_rollback           = 31
	Op_create_blob        = 34
	Op_open_blob          = 35
	Op_get_segment        = 36
	Op_put_segment        = 37
	Op_cancel_blob        = 38
	Op_close_blob         = 39
	Op_info_database      = 40
	Op_info_blob          = 43
	Op_info_transaction   = 42
	Op_batch_segments     = 44
	Op_que_events         = 48
	Op_cancel_events      = 49
	Op_commit_retaining   = 50
	Op_event              = 52
	Op_connect_request    = 53
	Op_open_blob2         = 56
	Op_create_blob2       = 57
	Op_get_slice          = 58
	Op_put_slice          = 59
	Op_slice_response     = 60
	Op_allocate_statement = 62
	Op_execute            = 63
	Op_exec_immediate     = 64
	Op_fetch              = 65
	Op_fetch_response     = 66
	Op_free_statement     = 67
	Op_prepare_statement  = 68
	Op_info_sql           = 70
	Op_dummy              = 71
	Op_seek_blob          = 61
	Op_execute2           = 76
	Op_sql_response       = 78
	Op_drop_database      = 81
	Op_service_attach     = 82
	Op_service_detach     = 83
	Op_service_info       = 84
	Op_service_start      = 85
	Op_rollback_retaining = 86
	// FB3
	Op_update_account_info  = 87
	Op_authenticate_user    = 88
	Op_partial              = 89
	Op_trusted_auth         = 90
	Op_cancel               = 91
	Op_cont_auth            = 92
	Op_ping                 = 93
	Op_accept_data          = 94
	Op_abort_aux_connection = 95
	Op_crypt                = 96
	Op_crypt_key_callback   = 97
	Op_cond_accept          = 98
	Op_batch_create         = 99
	Op_batch_msg            = 100
	Op_batch_exec           = 101
	Op_batch_rls            = 102
	Op_batch_cs             = 103
	Op_batch_regblob        = 104
	Op_batch_blob_stream    = 105
	Op_batch_set_bpb        = 106
	Op_repl_data            = 107
	Op_repl_req             = 108
	Op_batch_cancel         = 109
	Op_batch_sync           = 110
	Op_info_batch           = 111
	Op_inline_blob          = 114
)

const (
	ISOLATION_LEVEL_READ_COMMITED_LEGACY = iota
	ISOLATION_LEVEL_READ_COMMITED
	ISOLATION_LEVEL_REPEATABLE_READ
	ISOLATION_LEVEL_SERIALIZABLE
	ISOLATION_LEVEL_READ_COMMITED_RO
)

// Event
const (
	P_req_async             = 1 // Auxiliary asynchronous port
	EPB_version1            = 1
	Partner_identification  = 0
	Address_of_ast_routine  = 0
	Argument_to_ast_routine = 0
)

// op_cancel kind
const (
	Fb_cancel_disable = 1
	Fb_cancel_enable  = 2
	Fb_cancel_raise   = 3
	Fb_cancel_abort   = 4
)

type ShutdownMode byte

const (
	//ShutdownModeForce Wait for N seconds then shutdown
	ShutdownModeForce ShutdownMode = Isc_spb_prp_shutdown_db
	//ShutdownModeDenyNewAttachments Disable new attachments for N seconds then shutdown
	ShutdownModeDenyNewAttachments ShutdownMode = Isc_spb_prp_deny_new_attachments
	//ShutdownModeDenyNewTransactions Disable new transactions for N seconds then shutdown
	ShutdownModeDenyNewTransactions ShutdownMode = Isc_spb_prp_deny_new_transactions
)

type OperationMode byte

const (
	OperationModeNormal OperationMode = Isc_spb_prp_sm_normal
	OperationModeMulti  OperationMode = Isc_spb_prp_sm_multi
	OperationModeSingle OperationMode = Isc_spb_prp_sm_single
	OperationModeFull   OperationMode = Isc_spb_prp_sm_full
)

type ShutdownModeEx byte

const (
	//ShutdownModeExForce Wait for N seconds then shutdown
	ShutdownModeExForce ShutdownModeEx = Isc_spb_prp_force_shutdown
	//ShutdownModeExDenyNewAttachments Disable new attachments for N seconds then shutdown
	ShutdownModeExDenyNewAttachments ShutdownModeEx = Isc_spb_prp_attachments_shutdown
	//ShutdownModeExDenyNewTransactions Disable new transactions for N seconds then shutdown
	ShutdownModeExDenyNewTransactions ShutdownModeEx = Isc_spb_prp_transactions_shutdown
)

// op_free_statement p_sqlfree_option
const (
	DSQL_close     = 1
	DSQL_drop      = 2
	DSQL_unprepare = 4
)

const (
	CS_NONE    = 0
	CS_BINARY  = 1
	CS_dynamic = 127
)

const (
	Isc_blob_untyped = 0
	Isc_blob_text    = 1
)

const (
	BatchVersion1 = 1

	BatchTagMultiError     = 1
	BatchTagRecordCounts   = 2
	BatchTagBufferBytes    = 3
	BatchTagBlobPolicy     = 4
	BatchTagDetailedErrors = 5

	BatchInfoBufferBytes = 10
	BatchInfoDataBytes   = 11
	BatchInfoBlobsBytes  = 12
	BatchInfoBlobAlign   = 13
	BatchInfoBlobHeader  = 14

	BatchBlobNone     = 0
	BatchBlobIDEngine = 1
	BatchBlobIDUser   = 2
	BatchBlobStream   = 3

	BatchBlobSegHdrAlign = 2

	BatchExecuteFailed = -1
	BatchSuccessNoInfo = -2
)
const (
	BLOB_SUB_TYPE_BINARY = Isc_blob_untyped
	BLOB_SUB_TYPE_TEXT   = Isc_blob_text
)

const (
	SQL_TYPE_TEXT         = 452
	SQL_TYPE_VARYING      = 448
	SQL_TYPE_SHORT        = 500
	SQL_TYPE_LONG         = 496
	SQL_TYPE_FLOAT        = 482
	SQL_TYPE_DOUBLE       = 480
	SQL_TYPE_D_FLOAT      = 530
	SQL_TYPE_TIMESTAMP    = 510
	SQL_TYPE_BLOB         = 520
	SQL_TYPE_ARRAY        = 540
	SQL_TYPE_QUAD         = 550
	SQL_TYPE_TIME         = 560
	SQL_TYPE_DATE         = 570
	SQL_TYPE_INT64        = 580
	SQL_TYPE_INT128       = 32752
	SQL_TYPE_TIMESTAMP_TZ = 32754
	SQL_TYPE_TIME_TZ      = 32756
	SQL_TIMESTAMP_TZ_EX   = 32748
	SQL_TIME_TZ_EX        = 32750
	SQL_TYPE_DEC_FIXED    = 32758
	SQL_TYPE_DEC64        = 32760
	SQL_TYPE_DEC128       = 32762
	SQL_TYPE_BOOLEAN      = 32764
	SQL_TYPE_NULL         = 32766
)
