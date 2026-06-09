// Code generated from _tools/msgs DO NOT EDIT.
package fbproto

type fberror struct {
	message  string
	sqlstate string
}

const (
	Isc_arith_except = 335544321
	Isc_bad_dbkey = 335544322
	Isc_bad_db_format = 335544323
	Isc_bad_db_handle = 335544324
	Isc_bad_dpb_content = 335544325
	Isc_bad_dpb_form = 335544326
	Isc_bad_req_handle = 335544327
	Isc_bad_segstr_handle = 335544328
	Isc_bad_segstr_id = 335544329
	Isc_bad_tpb_content = 335544330
	Isc_bad_tpb_form = 335544331
	Isc_bad_trans_handle = 335544332
	Isc_bug_check = 335544333
	Isc_convert_error = 335544334
	Isc_db_corrupt = 335544335
	Isc_deadlock = 335544336
	Isc_excess_trans = 335544337
	Isc_from_no_match = 335544338
	Isc_infinap = 335544339
	Isc_infona = 335544340
	Isc_infunk = 335544341
	Isc_integ_fail = 335544342
	Isc_invalid_blr = 335544343
	Isc_io_error = 335544344
	Isc_lock_conflict = 335544345
	Isc_metadata_corrupt = 335544346
	Isc_not_valid = 335544347
	Isc_no_cur_rec = 335544348
	Isc_no_dup = 335544349
	Isc_no_finish = 335544350
	Isc_no_meta_update = 335544351
	Isc_no_priv = 335544352
	Isc_no_recon = 335544353
	Isc_no_record = 335544354
	Isc_no_segstr_close = 335544355
	Isc_obsolete_metadata = 335544356
	Isc_open_trans = 335544357
	Isc_port_len = 335544358
	Isc_read_only_field = 335544359
	Isc_read_only_rel = 335544360
	Isc_read_only_trans = 335544361
	Isc_read_only_view = 335544362
	Isc_req_no_trans = 335544363
	Isc_req_sync = 335544364
	Isc_req_wrong_db = 335544365
	Isc_segment = 335544366
	Isc_segstr_eof = 335544367
	Isc_segstr_no_op = 335544368
	Isc_segstr_no_read = 335544369
	Isc_segstr_no_trans = 335544370
	Isc_segstr_no_write = 335544371
	Isc_segstr_wrong_db = 335544372
	Isc_sys_request = 335544373
	Isc_stream_eof = 335544374
	Isc_unavailable = 335544375
	Isc_unres_rel = 335544376
	Isc_uns_ext = 335544377
	Isc_wish_list = 335544378
	Isc_wrong_ods = 335544379
	Isc_wronumarg = 335544380
	Isc_imp_exc = 335544381
	Isc_random = 335544382
	Isc_fatal_conflict = 335544383
	Isc_badblk = 335544384
	Isc_invpoolcl = 335544385
	Isc_nopoolids = 335544386
	Isc_relbadblk = 335544387
	Isc_blktoobig = 335544388
	Isc_bufexh = 335544389
	Isc_syntaxerr = 335544390
	Isc_bufinuse = 335544391
	Isc_bdbincon = 335544392
	Isc_reqinuse = 335544393
	Isc_badodsver = 335544394
	Isc_relnotdef = 335544395
	Isc_fldnotdef = 335544396
	Isc_dirtypage = 335544397
	Isc_waifortra = 335544398
	Isc_doubleloc = 335544399
	Isc_nodnotfnd = 335544400
	Isc_dupnodfnd = 335544401
	Isc_locnotmar = 335544402
	Isc_badpagtyp = 335544403
	Isc_corrupt = 335544404
	Isc_badpage = 335544405
	Isc_badindex = 335544406
	Isc_dbbnotzer = 335544407
	Isc_tranotzer = 335544408
	Isc_trareqmis = 335544409
	Isc_badhndcnt = 335544410
	Isc_wrotpbver = 335544411
	Isc_wroblrver = 335544412
	Isc_wrodpbver = 335544413
	Isc_blobnotsup = 335544414
	Isc_badrelation = 335544415
	Isc_nodetach = 335544416
	Isc_notremote = 335544417
	Isc_trainlim = 335544418
	Isc_notinlim = 335544419
	Isc_traoutsta = 335544420
	Isc_connect_reject = 335544421
	Isc_dbfile = 335544422
	Isc_orphan = 335544423
	Isc_no_lock_mgr = 335544424
	Isc_ctxinuse = 335544425
	Isc_ctxnotdef = 335544426
	Isc_datnotsup = 335544427
	Isc_badmsgnum = 335544428
	Isc_badparnum = 335544429
	Isc_virmemexh = 335544430
	Isc_blocking_signal = 335544431
	Isc_lockmanerr = 335544432
	Isc_journerr = 335544433
	Isc_keytoobig = 335544434
	Isc_nullsegkey = 335544435
	Isc_sqlerr = 335544436
	Isc_wrodynver = 335544437
	Isc_funnotdef = 335544438
	Isc_funmismat = 335544439
	Isc_bad_msg_vec = 335544440
	Isc_bad_detach = 335544441
	Isc_noargacc_read = 335544442
	Isc_noargacc_write = 335544443
	Isc_read_only = 335544444
	Isc_ext_err = 335544445
	Isc_non_updatable = 335544446
	Isc_no_rollback = 335544447
	Isc_bad_sec_info = 335544448
	Isc_invalid_sec_info = 335544449
	Isc_misc_interpreted = 335544450
	Isc_update_conflict = 335544451
	Isc_unlicensed = 335544452
	Isc_obj_in_use = 335544453
	Isc_nofilter = 335544454
	Isc_shadow_accessed = 335544455
	Isc_invalid_sdl = 335544456
	Isc_out_of_bounds = 335544457
	Isc_invalid_dimension = 335544458
	Isc_rec_in_limbo = 335544459
	Isc_shadow_missing = 335544460
	Isc_cant_validate = 335544461
	Isc_cant_start_journal = 335544462
	Isc_gennotdef = 335544463
	Isc_cant_start_logging = 335544464
	Isc_bad_segstr_type = 335544465
	Isc_foreign_key = 335544466
	Isc_high_minor = 335544467
	Isc_tra_state = 335544468
	Isc_trans_invalid = 335544469
	Isc_buf_invalid = 335544470
	Isc_indexnotdefined = 335544471
	Isc_login = 335544472
	Isc_invalid_bookmark = 335544473
	Isc_bad_lock_level = 335544474
	Isc_relation_lock = 335544475
	Isc_record_lock = 335544476
	Isc_max_idx = 335544477
	Isc_jrn_enable = 335544478
	Isc_old_failure = 335544479
	Isc_old_in_progress = 335544480
	Isc_old_no_space = 335544481
	Isc_no_wal_no_jrn = 335544482
	Isc_num_old_files = 335544483
	Isc_wal_file_open = 335544484
	Isc_bad_stmt_handle = 335544485
	Isc_wal_failure = 335544486
	Isc_walw_err = 335544487
	Isc_logh_small = 335544488
	Isc_logh_inv_version = 335544489
	Isc_logh_open_flag = 335544490
	Isc_logh_open_flag2 = 335544491
	Isc_logh_diff_dbname = 335544492
	Isc_logf_unexpected_eof = 335544493
	Isc_logr_incomplete = 335544494
	Isc_logr_header_small = 335544495
	Isc_logb_small = 335544496
	Isc_wal_illegal_attach = 335544497
	Isc_wal_invalid_wpb = 335544498
	Isc_wal_err_rollover = 335544499
	Isc_no_wal = 335544500
	Isc_drop_wal = 335544501
	Isc_stream_not_defined = 335544502
	Isc_wal_subsys_error = 335544503
	Isc_wal_subsys_corrupt = 335544504
	Isc_no_archive = 335544505
	Isc_shutinprog = 335544506
	Isc_range_in_use = 335544507
	Isc_range_not_found = 335544508
	Isc_charset_not_found = 335544509
	Isc_lock_timeout = 335544510
	Isc_prcnotdef = 335544511
	Isc_prcmismat = 335544512
	Isc_wal_bugcheck = 335544513
	Isc_wal_cant_expand = 335544514
	Isc_codnotdef = 335544515
	Isc_xcpnotdef = 335544516
	Isc_except = 335544517
	Isc_cache_restart = 335544518
	Isc_bad_lock_handle = 335544519
	Isc_jrn_present = 335544520
	Isc_wal_err_rollover2 = 335544521
	Isc_wal_err_logwrite = 335544522
	Isc_wal_err_jrn_comm = 335544523
	Isc_wal_err_expansion = 335544524
	Isc_wal_err_setup = 335544525
	Isc_wal_err_ww_sync = 335544526
	Isc_wal_err_ww_start = 335544527
	Isc_shutdown = 335544528
	Isc_existing_priv_mod = 335544529
	Isc_primary_key_ref = 335544530
	Isc_primary_key_notnull = 335544531
	Isc_ref_cnstrnt_notfound = 335544532
	Isc_foreign_key_notfound = 335544533
	Isc_ref_cnstrnt_update = 335544534
	Isc_check_cnstrnt_update = 335544535
	Isc_check_cnstrnt_del = 335544536
	Isc_integ_index_seg_del = 335544537
	Isc_integ_index_seg_mod = 335544538
	Isc_integ_index_del = 335544539
	Isc_integ_index_mod = 335544540
	Isc_check_trig_del = 335544541
	Isc_check_trig_update = 335544542
	Isc_cnstrnt_fld_del = 335544543
	Isc_cnstrnt_fld_rename = 335544544
	Isc_rel_cnstrnt_update = 335544545
	Isc_constaint_on_view = 335544546
	Isc_invld_cnstrnt_type = 335544547
	Isc_primary_key_exists = 335544548
	Isc_systrig_update = 335544549
	Isc_not_rel_owner = 335544550
	Isc_grant_obj_notfound = 335544551
	Isc_grant_fld_notfound = 335544552
	Isc_grant_nopriv = 335544553
	Isc_nonsql_security_rel = 335544554
	Isc_nonsql_security_fld = 335544555
	Isc_wal_cache_err = 335544556
	Isc_shutfail = 335544557
	Isc_check_constraint = 335544558
	Isc_bad_svc_handle = 335544559
	Isc_shutwarn = 335544560
	Isc_wrospbver = 335544561
	Isc_bad_spb_form = 335544562
	Isc_svcnotdef = 335544563
	Isc_no_jrn = 335544564
	Isc_transliteration_failed = 335544565
	Isc_start_cm_for_wal = 335544566
	Isc_wal_ovflow_log_required = 335544567
	Isc_text_subtype = 335544568
	Isc_dsql_error = 335544569
	Isc_dsql_command_err = 335544570
	Isc_dsql_constant_err = 335544571
	Isc_dsql_cursor_err = 335544572
	Isc_dsql_datatype_err = 335544573
	Isc_dsql_decl_err = 335544574
	Isc_dsql_cursor_update_err = 335544575
	Isc_dsql_cursor_open_err = 335544576
	Isc_dsql_cursor_close_err = 335544577
	Isc_dsql_field_err = 335544578
	Isc_dsql_internal_err = 335544579
	Isc_dsql_relation_err = 335544580
	Isc_dsql_procedure_err = 335544581
	Isc_dsql_request_err = 335544582
	Isc_dsql_sqlda_err = 335544583
	Isc_dsql_var_count_err = 335544584
	Isc_dsql_stmt_handle = 335544585
	Isc_dsql_function_err = 335544586
	Isc_dsql_blob_err = 335544587
	Isc_collation_not_found = 335544588
	Isc_collation_not_for_charset = 335544589
	Isc_dsql_dup_option = 335544590
	Isc_dsql_tran_err = 335544591
	Isc_dsql_invalid_array = 335544592
	Isc_dsql_max_arr_dim_exceeded = 335544593
	Isc_dsql_arr_range_error = 335544594
	Isc_dsql_trigger_err = 335544595
	Isc_dsql_subselect_err = 335544596
	Isc_dsql_crdb_prepare_err = 335544597
	Isc_specify_field_err = 335544598
	Isc_num_field_err = 335544599
	Isc_col_name_err = 335544600
	Isc_where_err = 335544601
	Isc_table_view_err = 335544602
	Isc_distinct_err = 335544603
	Isc_key_field_count_err = 335544604
	Isc_subquery_err = 335544605
	Isc_expression_eval_err = 335544606
	Isc_node_err = 335544607
	Isc_command_end_err = 335544608
	Isc_index_name = 335544609
	Isc_exception_name = 335544610
	Isc_field_name = 335544611
	Isc_token_err = 335544612
	Isc_union_err = 335544613
	Isc_dsql_construct_err = 335544614
	Isc_field_aggregate_err = 335544615
	Isc_field_ref_err = 335544616
	Isc_order_by_err = 335544617
	Isc_return_mode_err = 335544618
	Isc_extern_func_err = 335544619
	Isc_alias_conflict_err = 335544620
	Isc_procedure_conflict_error = 335544621
	Isc_relation_conflict_err = 335544622
	Isc_dsql_domain_err = 335544623
	Isc_idx_seg_err = 335544624
	Isc_node_name_err = 335544625
	Isc_table_name = 335544626
	Isc_proc_name = 335544627
	Isc_idx_create_err = 335544628
	Isc_wal_shadow_err = 335544629
	Isc_dependency = 335544630
	Isc_idx_key_err = 335544631
	Isc_dsql_file_length_err = 335544632
	Isc_dsql_shadow_number_err = 335544633
	Isc_dsql_token_unk_err = 335544634
	Isc_dsql_no_relation_alias = 335544635
	Isc_indexname = 335544636
	Isc_no_stream_plan = 335544637
	Isc_stream_twice = 335544638
	Isc_stream_not_found = 335544639
	Isc_collation_requires_text = 335544640
	Isc_dsql_domain_not_found = 335544641
	Isc_index_unused = 335544642
	Isc_dsql_self_join = 335544643
	Isc_stream_bof = 335544644
	Isc_stream_crack = 335544645
	Isc_db_or_file_exists = 335544646
	Isc_invalid_operator = 335544647
	Isc_conn_lost = 335544648
	Isc_bad_checksum = 335544649
	Isc_page_type_err = 335544650
	Isc_ext_readonly_err = 335544651
	Isc_sing_select_err = 335544652
	Isc_psw_attach = 335544653
	Isc_psw_start_trans = 335544654
	Isc_invalid_direction = 335544655
	Isc_dsql_var_conflict = 335544656
	Isc_dsql_no_blob_array = 335544657
	Isc_dsql_base_table = 335544658
	Isc_duplicate_base_table = 335544659
	Isc_view_alias = 335544660
	Isc_index_root_page_full = 335544661
	Isc_dsql_blob_type_unknown = 335544662
	Isc_req_max_clones_exceeded = 335544663
	Isc_dsql_duplicate_spec = 335544664
	Isc_unique_key_violation = 335544665
	Isc_srvr_version_too_old = 335544666
	Isc_drdb_completed_with_errs = 335544667
	Isc_dsql_procedure_use_err = 335544668
	Isc_dsql_count_mismatch = 335544669
	Isc_blob_idx_err = 335544670
	Isc_array_idx_err = 335544671
	Isc_key_field_err = 335544672
	Isc_no_delete = 335544673
	Isc_del_last_field = 335544674
	Isc_sort_err = 335544675
	Isc_sort_mem_err = 335544676
	Isc_version_err = 335544677
	Isc_inval_key_posn = 335544678
	Isc_no_segments_err = 335544679
	Isc_crrp_data_err = 335544680
	Isc_rec_size_err = 335544681
	Isc_dsql_field_ref = 335544682
	Isc_req_depth_exceeded = 335544683
	Isc_no_field_access = 335544684
	Isc_no_dbkey = 335544685
	Isc_jrn_format_err = 335544686
	Isc_jrn_file_full = 335544687
	Isc_dsql_open_cursor_request = 335544688
	Isc_ib_error = 335544689
	Isc_cache_redef = 335544690
	Isc_cache_too_small = 335544691
	Isc_log_redef = 335544692
	Isc_log_too_small = 335544693
	Isc_partition_too_small = 335544694
	Isc_partition_not_supp = 335544695
	Isc_log_length_spec = 335544696
	Isc_precision_err = 335544697
	Isc_scale_nogt = 335544698
	Isc_expec_short = 335544699
	Isc_expec_long = 335544700
	Isc_expec_ushort = 335544701
	Isc_escape_invalid = 335544702
	Isc_svcnoexe = 335544703
	Isc_net_lookup_err = 335544704
	Isc_service_unknown = 335544705
	Isc_host_unknown = 335544706
	Isc_grant_nopriv_on_base = 335544707
	Isc_dyn_fld_ambiguous = 335544708
	Isc_dsql_agg_ref_err = 335544709
	Isc_complex_view = 335544710
	Isc_unprepared_stmt = 335544711
	Isc_expec_positive = 335544712
	Isc_dsql_sqlda_value_err = 335544713
	Isc_invalid_array_id = 335544714
	Isc_extfile_uns_op = 335544715
	Isc_svc_in_use = 335544716
	Isc_err_stack_limit = 335544717
	Isc_invalid_key = 335544718
	Isc_net_init_error = 335544719
	Isc_loadlib_failure = 335544720
	Isc_network_error = 335544721
	Isc_net_connect_err = 335544722
	Isc_net_connect_listen_err = 335544723
	Isc_net_event_connect_err = 335544724
	Isc_net_event_listen_err = 335544725
	Isc_net_read_err = 335544726
	Isc_net_write_err = 335544727
	Isc_integ_index_deactivate = 335544728
	Isc_integ_deactivate_primary = 335544729
	Isc_cse_not_supported = 335544730
	Isc_tra_must_sweep = 335544731
	Isc_unsupported_network_drive = 335544732
	Isc_io_create_err = 335544733
	Isc_io_open_err = 335544734
	Isc_io_close_err = 335544735
	Isc_io_read_err = 335544736
	Isc_io_write_err = 335544737
	Isc_io_delete_err = 335544738
	Isc_io_access_err = 335544739
	Isc_udf_exception = 335544740
	Isc_lost_db_connection = 335544741
	Isc_no_write_user_priv = 335544742
	Isc_token_too_long = 335544743
	Isc_max_att_exceeded = 335544744
	Isc_login_same_as_role_name = 335544745
	Isc_reftable_requires_pk = 335544746
	Isc_usrname_too_long = 335544747
	Isc_password_too_long = 335544748
	Isc_usrname_required = 335544749
	Isc_password_required = 335544750
	Isc_bad_protocol = 335544751
	Isc_dup_usrname_found = 335544752
	Isc_usrname_not_found = 335544753
	Isc_error_adding_sec_record = 335544754
	Isc_error_modifying_sec_record = 335544755
	Isc_error_deleting_sec_record = 335544756
	Isc_error_updating_sec_db = 335544757
	Isc_sort_rec_size_err = 335544758
	Isc_bad_default_value = 335544759
	Isc_invalid_clause = 335544760
	Isc_too_many_handles = 335544761
	Isc_optimizer_blk_exc = 335544762
	Isc_invalid_string_constant = 335544763
	Isc_transitional_date = 335544764
	Isc_read_only_database = 335544765
	Isc_must_be_dialect_2_and_up = 335544766
	Isc_blob_filter_exception = 335544767
	Isc_exception_access_violation = 335544768
	Isc_exception_datatype_missalignment = 335544769
	Isc_exception_array_bounds_exceeded = 335544770
	Isc_exception_float_denormal_operand = 335544771
	Isc_exception_float_divide_by_zero = 335544772
	Isc_exception_float_inexact_result = 335544773
	Isc_exception_float_invalid_operand = 335544774
	Isc_exception_float_overflow = 335544775
	Isc_exception_float_stack_check = 335544776
	Isc_exception_float_underflow = 335544777
	Isc_exception_integer_divide_by_zero = 335544778
	Isc_exception_integer_overflow = 335544779
	Isc_exception_unknown = 335544780
	Isc_exception_stack_overflow = 335544781
	Isc_exception_sigsegv = 335544782
	Isc_exception_sigill = 335544783
	Isc_exception_sigbus = 335544784
	Isc_exception_sigfpe = 335544785
	Isc_ext_file_delete = 335544786
	Isc_ext_file_modify = 335544787
	Isc_adm_task_denied = 335544788
	Isc_extract_input_mismatch = 335544789
	Isc_insufficient_svc_privileges = 335544790
	Isc_file_in_use = 335544791
	Isc_service_att_err = 335544792
	Isc_ddl_not_allowed_by_db_sql_dial = 335544793
	Isc_cancelled = 335544794
	Isc_unexp_spb_form = 335544795
	Isc_sql_dialect_datatype_unsupport = 335544796
	Isc_svcnouser = 335544797
	Isc_depend_on_uncommitted_rel = 335544798
	Isc_svc_name_missing = 335544799
	Isc_too_many_contexts = 335544800
	Isc_datype_notsup = 335544801
	Isc_dialect_reset_warning = 335544802
	Isc_dialect_not_changed = 335544803
	Isc_database_create_failed = 335544804
	Isc_inv_dialect_specified = 335544805
	Isc_valid_db_dialects = 335544806
	Isc_sqlwarn = 335544807
	Isc_dtype_renamed = 335544808
	Isc_extern_func_dir_error = 335544809
	Isc_date_range_exceeded = 335544810
	Isc_inv_client_dialect_specified = 335544811
	Isc_valid_client_dialects = 335544812
	Isc_optimizer_between_err = 335544813
	Isc_service_not_supported = 335544814
	Isc_generator_name = 335544815
	Isc_udf_name = 335544816
	Isc_bad_limit_param = 335544817
	Isc_bad_skip_param = 335544818
	Isc_io_32bit_exceeded_err = 335544819
	Isc_invalid_savepoint = 335544820
	Isc_dsql_column_pos_err = 335544821
	Isc_dsql_agg_where_err = 335544822
	Isc_dsql_agg_group_err = 335544823
	Isc_dsql_agg_column_err = 335544824
	Isc_dsql_agg_having_err = 335544825
	Isc_dsql_agg_nested_err = 335544826
	Isc_exec_sql_invalid_arg = 335544827
	Isc_exec_sql_invalid_req = 335544828
	Isc_exec_sql_invalid_var = 335544829
	Isc_exec_sql_max_call_exceeded = 335544830
	Isc_conf_access_denied = 335544831
	Isc_wrong_backup_state = 335544832
	Isc_wal_backup_err = 335544833
	Isc_cursor_not_open = 335544834
	Isc_bad_shutdown_mode = 335544835
	Isc_concat_overflow = 335544836
	Isc_bad_substring_offset = 335544837
	Isc_foreign_key_target_doesnt_exist = 335544838
	Isc_foreign_key_references_present = 335544839
	Isc_no_update = 335544840
	Isc_cursor_already_open = 335544841
	Isc_stack_trace = 335544842
	Isc_ctx_var_not_found = 335544843
	Isc_ctx_namespace_invalid = 335544844
	Isc_ctx_too_big = 335544845
	Isc_ctx_bad_argument = 335544846
	Isc_identifier_too_long = 335544847
	Isc_except2 = 335544848
	Isc_malformed_string = 335544849
	Isc_prc_out_param_mismatch = 335544850
	Isc_command_end_err2 = 335544851
	Isc_partner_idx_incompat_type = 335544852
	Isc_bad_substring_length = 335544853
	Isc_charset_not_installed = 335544854
	Isc_collation_not_installed = 335544855
	Isc_att_shutdown = 335544856
	Isc_blobtoobig = 335544857
	Isc_must_have_phys_field = 335544858
	Isc_invalid_time_precision = 335544859
	Isc_blob_convert_error = 335544860
	Isc_array_convert_error = 335544861
	Isc_record_lock_not_supp = 335544862
	Isc_partner_idx_not_found = 335544863
	Isc_tra_num_exc = 335544864
	Isc_field_disappeared = 335544865
	Isc_met_wrong_gtt_scope = 335544866
	Isc_subtype_for_internal_use = 335544867
	Isc_illegal_prc_type = 335544868
	Isc_invalid_sort_datatype = 335544869
	Isc_collation_name = 335544870
	Isc_domain_name = 335544871
	Isc_domnotdef = 335544872
	Isc_array_max_dimensions = 335544873
	Isc_max_db_per_trans_allowed = 335544874
	Isc_bad_debug_format = 335544875
	Isc_bad_proc_BLR = 335544876
	Isc_key_too_big = 335544877
	Isc_concurrent_transaction = 335544878
	Isc_not_valid_for_var = 335544879
	Isc_not_valid_for = 335544880
	Isc_need_difference = 335544881
	Isc_long_login = 335544882
	Isc_fldnotdef2 = 335544883
	Isc_invalid_similar_pattern = 335544884
	Isc_bad_teb_form = 335544885
	Isc_tpb_multiple_txn_isolation = 335544886
	Isc_tpb_reserv_before_table = 335544887
	Isc_tpb_multiple_spec = 335544888
	Isc_tpb_option_without_rc = 335544889
	Isc_tpb_conflicting_options = 335544890
	Isc_tpb_reserv_missing_tlen = 335544891
	Isc_tpb_reserv_long_tlen = 335544892
	Isc_tpb_reserv_missing_tname = 335544893
	Isc_tpb_reserv_corrup_tlen = 335544894
	Isc_tpb_reserv_null_tlen = 335544895
	Isc_tpb_reserv_relnotfound = 335544896
	Isc_tpb_reserv_baserelnotfound = 335544897
	Isc_tpb_missing_len = 335544898
	Isc_tpb_missing_value = 335544899
	Isc_tpb_corrupt_len = 335544900
	Isc_tpb_null_len = 335544901
	Isc_tpb_overflow_len = 335544902
	Isc_tpb_invalid_value = 335544903
	Isc_tpb_reserv_stronger_wng = 335544904
	Isc_tpb_reserv_stronger = 335544905
	Isc_tpb_reserv_max_recursion = 335544906
	Isc_tpb_reserv_virtualtbl = 335544907
	Isc_tpb_reserv_systbl = 335544908
	Isc_tpb_reserv_temptbl = 335544909
	Isc_tpb_readtxn_after_writelock = 335544910
	Isc_tpb_writelock_after_readtxn = 335544911
	Isc_time_range_exceeded = 335544912
	Isc_datetime_range_exceeded = 335544913
	Isc_string_truncation = 335544914
	Isc_blob_truncation = 335544915
	Isc_numeric_out_of_range = 335544916
	Isc_shutdown_timeout = 335544917
	Isc_att_handle_busy = 335544918
	Isc_bad_udf_freeit = 335544919
	Isc_eds_provider_not_found = 335544920
	Isc_eds_connection = 335544921
	Isc_eds_preprocess = 335544922
	Isc_eds_stmt_expected = 335544923
	Isc_eds_prm_name_expected = 335544924
	Isc_eds_unclosed_comment = 335544925
	Isc_eds_statement = 335544926
	Isc_eds_input_prm_mismatch = 335544927
	Isc_eds_output_prm_mismatch = 335544928
	Isc_eds_input_prm_not_set = 335544929
	Isc_too_big_blr = 335544930
	Isc_montabexh = 335544931
	Isc_modnotfound = 335544932
	Isc_nothing_to_cancel = 335544933
	Isc_ibutil_not_loaded = 335544934
	Isc_circular_computed = 335544935
	Isc_psw_db_error = 335544936
	Isc_invalid_type_datetime_op = 335544937
	Isc_onlycan_add_timetodate = 335544938
	Isc_onlycan_add_datetotime = 335544939
	Isc_onlycansub_tstampfromtstamp = 335544940
	Isc_onlyoneop_mustbe_tstamp = 335544941
	Isc_invalid_extractpart_time = 335544942
	Isc_invalid_extractpart_date = 335544943
	Isc_invalidarg_extract = 335544944
	Isc_sysf_argmustbe_exact = 335544945
	Isc_sysf_argmustbe_exact_or_fp = 335544946
	Isc_sysf_argviolates_uuidtype = 335544947
	Isc_sysf_argviolates_uuidlen = 335544948
	Isc_sysf_argviolates_uuidfmt = 335544949
	Isc_sysf_argviolates_guidigits = 335544950
	Isc_sysf_invalid_addpart_time = 335544951
	Isc_sysf_invalid_add_datetime = 335544952
	Isc_sysf_invalid_addpart_dtime = 335544953
	Isc_sysf_invalid_add_dtime_rc = 335544954
	Isc_sysf_invalid_diff_dtime = 335544955
	Isc_sysf_invalid_timediff = 335544956
	Isc_sysf_invalid_tstamptimediff = 335544957
	Isc_sysf_invalid_datetimediff = 335544958
	Isc_sysf_invalid_diffpart = 335544959
	Isc_sysf_argmustbe_positive = 335544960
	Isc_sysf_basemustbe_positive = 335544961
	Isc_sysf_argnmustbe_nonneg = 335544962
	Isc_sysf_argnmustbe_positive = 335544963
	Isc_sysf_invalid_zeropowneg = 335544964
	Isc_sysf_invalid_negpowfp = 335544965
	Isc_sysf_invalid_scale = 335544966
	Isc_sysf_argmustbe_nonneg = 335544967
	Isc_sysf_binuuid_mustbe_str = 335544968
	Isc_sysf_binuuid_wrongsize = 335544969
	Isc_missing_required_spb = 335544970
	Isc_net_server_shutdown = 335544971
	Isc_bad_conn_str = 335544972
	Isc_bad_epb_form = 335544973
	Isc_no_threads = 335544974
	Isc_net_event_connect_timeout = 335544975
	Isc_sysf_argmustbe_nonzero = 335544976
	Isc_sysf_argmustbe_range_inc1_1 = 335544977
	Isc_sysf_argmustbe_gteq_one = 335544978
	Isc_sysf_argmustbe_range_exc1_1 = 335544979
	Isc_internal_rejected_params = 335544980
	Isc_sysf_fp_overflow = 335544981
	Isc_udf_fp_overflow = 335544982
	Isc_udf_fp_nan = 335544983
	Isc_instance_conflict = 335544984
	Isc_out_of_temp_space = 335544985
	Isc_eds_expl_tran_ctrl = 335544986
	Isc_no_trusted_spb = 335544987
	Isc_package_name = 335544988
	Isc_cannot_make_not_null = 335544989
	Isc_feature_removed = 335544990
	Isc_view_name = 335544991
	Isc_lock_dir_access = 335544992
	Isc_invalid_fetch_option = 335544993
	Isc_bad_fun_BLR = 335544994
	Isc_func_pack_not_implemented = 335544995
	Isc_proc_pack_not_implemented = 335544996
	Isc_eem_func_not_returned = 335544997
	Isc_eem_proc_not_returned = 335544998
	Isc_eem_trig_not_returned = 335544999
	Isc_eem_bad_plugin_ver = 335545000
	Isc_eem_engine_notfound = 335545001
	Isc_attachment_in_use = 335545002
	Isc_transaction_in_use = 335545003
	Isc_pman_cannot_load_plugin = 335545004
	Isc_pman_module_notfound = 335545005
	Isc_pman_entrypoint_notfound = 335545006
	Isc_pman_module_bad = 335545007
	Isc_pman_plugin_notfound = 335545008
	Isc_sysf_invalid_trig_namespace = 335545009
	Isc_unexpected_null = 335545010
	Isc_type_notcompat_blob = 335545011
	Isc_invalid_date_val = 335545012
	Isc_invalid_time_val = 335545013
	Isc_invalid_timestamp_val = 335545014
	Isc_invalid_index_val = 335545015
	Isc_formatted_exception = 335545016
	Isc_async_active = 335545017
	Isc_private_function = 335545018
	Isc_private_procedure = 335545019
	Isc_request_outdated = 335545020
	Isc_bad_events_handle = 335545021
	Isc_cannot_copy_stmt = 335545022
	Isc_invalid_boolean_usage = 335545023
	Isc_sysf_argscant_both_be_zero = 335545024
	Isc_spb_no_id = 335545025
	Isc_ee_blr_mismatch_null = 335545026
	Isc_ee_blr_mismatch_length = 335545027
	Isc_ss_out_of_bounds = 335545028
	Isc_missing_data_structures = 335545029
	Isc_protect_sys_tab = 335545030
	Isc_libtommath_generic = 335545031
	Isc_wroblrver2 = 335545032
	Isc_trunc_limits = 335545033
	Isc_info_access = 335545034
	Isc_svc_no_stdin = 335545035
	Isc_svc_start_failed = 335545036
	Isc_svc_no_switches = 335545037
	Isc_svc_bad_size = 335545038
	Isc_no_crypt_plugin = 335545039
	Isc_cp_name_too_long = 335545040
	Isc_cp_process_active = 335545041
	Isc_cp_already_crypted = 335545042
	Isc_decrypt_error = 335545043
	Isc_no_providers = 335545044
	Isc_null_spb = 335545045
	Isc_max_args_exceeded = 335545046
	Isc_ee_blr_mismatch_names_count = 335545047
	Isc_ee_blr_mismatch_name_not_found = 335545048
	Isc_bad_result_set = 335545049
	Isc_wrong_message_length = 335545050
	Isc_no_output_format = 335545051
	Isc_item_finish = 335545052
	Isc_miss_config = 335545053
	Isc_conf_line = 335545054
	Isc_conf_include = 335545055
	Isc_include_depth = 335545056
	Isc_include_miss = 335545057
	Isc_protect_ownership = 335545058
	Isc_badvarnum = 335545059
	Isc_sec_context = 335545060
	Isc_multi_segment = 335545061
	Isc_login_changed = 335545062
	Isc_auth_handshake_limit = 335545063
	Isc_wirecrypt_incompatible = 335545064
	Isc_miss_wirecrypt = 335545065
	Isc_wirecrypt_key = 335545066
	Isc_wirecrypt_plugin = 335545067
	Isc_secdb_name = 335545068
	Isc_auth_data = 335545069
	Isc_auth_datalength = 335545070
	Isc_info_unprepared_stmt = 335545071
	Isc_idx_key_value = 335545072
	Isc_forupdate_virtualtbl = 335545073
	Isc_forupdate_systbl = 335545074
	Isc_forupdate_temptbl = 335545075
	Isc_cant_modify_sysobj = 335545076
	Isc_server_misconfigured = 335545077
	Isc_alter_role = 335545078
	Isc_map_already_exists = 335545079
	Isc_map_not_exists = 335545080
	Isc_map_load = 335545081
	Isc_map_aster = 335545082
	Isc_map_multi = 335545083
	Isc_map_undefined = 335545084
	Isc_baddpb_damaged_mode = 335545085
	Isc_baddpb_buffers_range = 335545086
	Isc_baddpb_temp_buffers = 335545087
	Isc_map_nodb = 335545088
	Isc_map_notable = 335545089
	Isc_miss_trusted_role = 335545090
	Isc_set_invalid_role = 335545091
	Isc_cursor_not_positioned = 335545092
	Isc_dup_attribute = 335545093
	Isc_dyn_no_priv = 335545094
	Isc_dsql_cant_grant_option = 335545095
	Isc_read_conflict = 335545096
	Isc_crdb_load = 335545097
	Isc_crdb_nodb = 335545098
	Isc_crdb_notable = 335545099
	Isc_interface_version_too_old = 335545100
	Isc_fun_param_mismatch = 335545101
	Isc_savepoint_backout_err = 335545102
	Isc_domain_primary_key_notnull = 335545103
	Isc_invalid_attachment_charset = 335545104
	Isc_map_down = 335545105
	Isc_login_error = 335545106
	Isc_already_opened = 335545107
	Isc_bad_crypt_key = 335545108
	Isc_encrypt_error = 335545109
	Isc_max_idx_depth = 335545110
	Isc_wrong_prvlg = 335545111
	Isc_miss_prvlg = 335545112
	Isc_crypt_checksum = 335545113
	Isc_not_dba = 335545114
	Isc_no_cursor = 335545115
	Isc_dsql_window_incompat_frames = 335545116
	Isc_dsql_window_range_multi_key = 335545117
	Isc_dsql_window_range_inv_key_type = 335545118
	Isc_dsql_window_frame_value_inv_type = 335545119
	Isc_window_frame_value_invalid = 335545120
	Isc_dsql_window_not_found = 335545121
	Isc_dsql_window_cant_overr_part = 335545122
	Isc_dsql_window_cant_overr_order = 335545123
	Isc_dsql_window_cant_overr_frame = 335545124
	Isc_dsql_window_duplicate = 335545125
	Isc_sql_too_long = 335545126
	Isc_cfg_stmt_timeout = 335545127
	Isc_att_stmt_timeout = 335545128
	Isc_req_stmt_timeout = 335545129
	Isc_att_shut_killed = 335545130
	Isc_att_shut_idle = 335545131
	Isc_att_shut_db_down = 335545132
	Isc_att_shut_engine = 335545133
	Isc_overriding_without_identity = 335545134
	Isc_overriding_system_invalid = 335545135
	Isc_overriding_user_invalid = 335545136
	Isc_overriding_missing = 335545137
	Isc_decprecision_err = 335545138
	Isc_decfloat_divide_by_zero = 335545139
	Isc_decfloat_inexact_result = 335545140
	Isc_decfloat_invalid_operation = 335545141
	Isc_decfloat_overflow = 335545142
	Isc_decfloat_underflow = 335545143
	Isc_subfunc_notdef = 335545144
	Isc_subproc_notdef = 335545145
	Isc_subfunc_signat = 335545146
	Isc_subproc_signat = 335545147
	Isc_subfunc_defvaldecl = 335545148
	Isc_subproc_defvaldecl = 335545149
	Isc_subfunc_not_impl = 335545150
	Isc_subproc_not_impl = 335545151
	Isc_sysf_invalid_hash_algorithm = 335545152
	Isc_expression_eval_index = 335545153
	Isc_invalid_decfloat_trap = 335545154
	Isc_invalid_decfloat_round = 335545155
	Isc_sysf_invalid_first_last_part = 335545156
	Isc_sysf_invalid_date_timestamp = 335545157
	Isc_precision_err2 = 335545158
	Isc_bad_batch_handle = 335545159
	Isc_intl_char = 335545160
	Isc_null_block = 335545161
	Isc_mixed_info = 335545162
	Isc_unknown_info = 335545163
	Isc_bpb_version = 335545164
	Isc_user_manager = 335545165
	Isc_icu_entrypoint = 335545166
	Isc_icu_library = 335545167
	Isc_metadata_name = 335545168
	Isc_tokens_parse = 335545169
	Isc_iconv_open = 335545170
	Isc_batch_compl_range = 335545171
	Isc_batch_compl_detail = 335545172
	Isc_deflate_init = 335545173
	Isc_inflate_init = 335545174
	Isc_big_segment = 335545175
	Isc_batch_policy = 335545176
	Isc_batch_defbpb = 335545177
	Isc_batch_align = 335545178
	Isc_multi_segment_dup = 335545179
	Isc_non_plugin_protocol = 335545180
	Isc_message_format = 335545181
	Isc_batch_param_version = 335545182
	Isc_batch_msg_long = 335545183
	Isc_batch_open = 335545184
	Isc_batch_type = 335545185
	Isc_batch_param = 335545186
	Isc_batch_blobs = 335545187
	Isc_batch_blob_append = 335545188
	Isc_batch_stream_align = 335545189
	Isc_batch_rpt_blob = 335545190
	Isc_batch_blob_buf = 335545191
	Isc_batch_small_data = 335545192
	Isc_batch_cont_bpb = 335545193
	Isc_batch_big_bpb = 335545194
	Isc_batch_big_segment = 335545195
	Isc_batch_big_seg2 = 335545196
	Isc_batch_blob_id = 335545197
	Isc_batch_too_big = 335545198
	Isc_num_literal = 335545199
	Isc_map_event = 335545200
	Isc_map_overflow = 335545201
	Isc_hdr_overflow = 335545202
	Isc_vld_plugins = 335545203
	Isc_db_crypt_key = 335545204
	Isc_no_keyholder_plugin = 335545205
	Isc_ses_reset_err = 335545206
	Isc_ses_reset_open_trans = 335545207
	Isc_ses_reset_warn = 335545208
	Isc_ses_reset_tran_rollback = 335545209
	Isc_plugin_name = 335545210
	Isc_parameter_name = 335545211
	Isc_file_starting_page_err = 335545212
	Isc_invalid_timezone_offset = 335545213
	Isc_invalid_timezone_region = 335545214
	Isc_invalid_timezone_id = 335545215
	Isc_tom_decode64len = 335545216
	Isc_tom_strblob = 335545217
	Isc_tom_reg = 335545218
	Isc_tom_algorithm = 335545219
	Isc_tom_mode_miss = 335545220
	Isc_tom_mode_bad = 335545221
	Isc_tom_no_mode = 335545222
	Isc_tom_iv_miss = 335545223
	Isc_tom_no_iv = 335545224
	Isc_tom_ctrtype_bad = 335545225
	Isc_tom_no_ctrtype = 335545226
	Isc_tom_ctr_big = 335545227
	Isc_tom_no_ctr = 335545228
	Isc_tom_iv_length = 335545229
	Isc_tom_error = 335545230
	Isc_tom_yarrow_start = 335545231
	Isc_tom_yarrow_setup = 335545232
	Isc_tom_init_mode = 335545233
	Isc_tom_crypt_mode = 335545234
	Isc_tom_decrypt_mode = 335545235
	Isc_tom_init_cip = 335545236
	Isc_tom_crypt_cip = 335545237
	Isc_tom_decrypt_cip = 335545238
	Isc_tom_setup_cip = 335545239
	Isc_tom_setup_chacha = 335545240
	Isc_tom_encode = 335545241
	Isc_tom_decode = 335545242
	Isc_tom_rsa_import = 335545243
	Isc_tom_oaep = 335545244
	Isc_tom_hash_bad = 335545245
	Isc_tom_rsa_make = 335545246
	Isc_tom_rsa_export = 335545247
	Isc_tom_rsa_sign = 335545248
	Isc_tom_rsa_verify = 335545249
	Isc_tom_chacha_key = 335545250
	Isc_bad_repl_handle = 335545251
	Isc_tra_snapshot_does_not_exist = 335545252
	Isc_eds_input_prm_not_used = 335545253
	Isc_effective_user = 335545254
	Isc_invalid_time_zone_bind = 335545255
	Isc_invalid_decfloat_bind = 335545256
	Isc_odd_hex_len = 335545257
	Isc_invalid_hex_digit = 335545258
	Isc_bind_err = 335545259
	Isc_bind_statement = 335545260
	Isc_bind_convert = 335545261
	Isc_cannot_update_old_blob = 335545262
	Isc_cannot_read_new_blob = 335545263
	Isc_dyn_no_create_priv = 335545264
	Isc_suspend_without_returns = 335545265
	Isc_truncate_warn = 335545266
	Isc_truncate_monitor = 335545267
	Isc_truncate_context = 335545268
	Isc_merge_dup_update = 335545269
	Isc_wrong_page = 335545270
	Isc_repl_error = 335545271
	Isc_ses_reset_failed = 335545272
	Isc_block_size = 335545273
	Isc_tom_key_length = 335545274
	Isc_inf_invalid_args = 335545275
	Isc_sysf_invalid_null_empty = 335545276
	Isc_bad_loctab_num = 335545277
	Isc_quoted_str_bad = 335545278
	Isc_quoted_str_miss = 335545279
	Isc_wrong_shmem_ver = 335545280
	Isc_wrong_shmem_bitness = 335545281
	Isc_wrong_proc_plan = 335545282
	Isc_invalid_blob_util_handle = 335545283
	Isc_bad_temp_blob_id = 335545284
	Isc_ods_upgrade_err = 335545285
	Isc_bad_par_workers = 335545286
	Isc_idx_expr_not_found = 335545287
	Isc_idx_cond_not_found = 335545288
	Isc_uninitialized_var = 335545289
	Isc_param_not_exist = 335545290
	Isc_param_no_default_not_specified = 335545291
	Isc_param_multiple_assignments = 335545292
	Isc_invalid_date_format = 335545293
	Isc_invalid_raw_string_in_date_format = 335545294
	Isc_invalid_data_type_for_date_format = 335545295
	Isc_incompatible_date_format_with_current_date_type = 335545296
	Isc_value_for_pattern_is_out_of_range = 335545297
	Isc_month_name_mismatch = 335545298
	Isc_incorrect_hours_period = 335545299
	Isc_data_for_format_is_exhausted = 335545300
	Isc_trailing_part_of_string = 335545301
	Isc_pattern_cant_be_used_without_other_pattern = 335545302
	Isc_pattern_cant_be_used_without_other_pattern_and_vice_versa = 335545303
	Isc_incompatible_format_patterns = 335545304
	Isc_only_one_pattern_can_be_used = 335545305
	Isc_can_not_use_same_pattern_twice = 335545306
	Isc_sysf_invalid_gen_uuid_version = 335545307
	Isc_sweep_unable_to_run = 335545308
	Isc_sweep_concurrent_instance = 335545309
	Isc_sweep_read_only = 335545310
	Isc_sweep_attach_no_cleanup = 335545311
	Isc_invalid_timezone_region_or_displacement = 335545312
	Isc_argmustbe_exact_range_for = 335545313
	Isc_range_for_by_should_be_positive = 335545314
	Isc_missing_value_for_format_pattern = 335545315
	Isc_invalid_name = 335545316
	Isc_invalid_unqualified_name_list = 335545317
	Isc_no_user_att_while_restore = 335545318
	Isc_genseq_stepmustbe_nonzero = 335545319
	Isc_argmustbe_exact_function = 335545320
	Isc_bad_ext_file = 335546320
	Isc_wrong_adp_fields_def = 335546321
	Isc_wrong_adp_field_name = 335546322
	Isc_wrong_adp_field_type = 335546323
	Isc_bad_ext_record = 335546324
	Isc_bad_adp_type = 335546325
	Isc_dir_resolve_error = 335546326
	Isc_read_file_error = 335546327
	Isc_create_file_error = 335546328
	Isc_delete_file_error = 335546329
	Isc_blob_io_error = 335546330
	Isc_cant_connect_to_ldap = 335546331
	Isc_cant_bind_to_ldap = 335546332
	Isc_no_ldap_init = 335546333
	Isc_sysf_ldap_attr = 335546334
	Isc_no_ext_file = 335546335
	Isc_no_gss_init = 335546336
	Isc_hash_error = 335546337
	Isc_policy_already_exists = 335546338
	Isc_policy_not_exists = 335546339
	Isc_user_not_exists = 335546340
	Isc_policy_is_granted = 335546341
	Isc_bad_trig_BLR = 335546342
	Isc_not_rdb = 335546343
	Isc_trace_write_error = 335546345
	Isc_no_granted_policy = 335546346
	Isc_regexp_substr_error = 335546347
	Isc_ts_file_exists = 335546348
	Isc_tablespace_name = 335546349
	Isc_ts_file_not_exists = 335546350
	Isc_secdb_coonection_error = 335546351
	Isc_policies_reading_error = 335546352
	Isc_get_policies_error = 335546353
	Isc_att_suspended = 335546354
	Isc_suspend_privileges_error = 335546355
	Isc_trace_buffer_limit = 335546356
	Isc_pman_hash_incorrect = 335546357
	Isc_json_table_plan_missing_name = 335546358
	Isc_json_table_same_names = 335546359
	Isc_json_table_plan_same_names = 335546360
	Isc_json_table_plan_double_appearance = 335546361
	Isc_json_table_plan_wrong_level = 335546362
	Isc_json_table_invalid_plan = 335546363
	Isc_json_table_plan_wrong_name = 335546364
	Isc_foreign_server = 335546369
	Isc_foreign_provider_not_found = 335546370
	Isc_change_forced_write_mode_with_connections = 335546372
	Isc_license_not_found = 335546373
	Isc_feature_not_available = 335546374
	Isc_repl_mispos = 335546375
	Isc_missing_trace_plugin = 335546378
	Isc_repl_recover_err = 335546379
	Isc_user_policy_reading_error = 335546380
	Isc_get_user_policy_error = 335546381
	Isc_gfix_db_name = 335740929
	Isc_gfix_invalid_sw = 335740930
	Isc_gfix_incmp_sw = 335740932
	Isc_gfix_replay_req = 335740933
	Isc_gfix_pgbuf_req = 335740934
	Isc_gfix_val_req = 335740935
	Isc_gfix_pval_req = 335740936
	Isc_gfix_trn_req = 335740937
	Isc_gfix_full_req = 335740940
	Isc_gfix_usrname_req = 335740941
	Isc_gfix_pass_req = 335740942
	Isc_gfix_subs_name = 335740943
	Isc_gfix_wal_req = 335740944
	Isc_gfix_sec_req = 335740945
	Isc_gfix_nval_req = 335740946
	Isc_gfix_type_shut = 335740947
	Isc_gfix_retry = 335740948
	Isc_gfix_retry_db = 335740951
	Isc_gfix_exceed_max = 335740991
	Isc_gfix_corrupt_pool = 335740992
	Isc_gfix_mem_exhausted = 335740993
	Isc_gfix_bad_pool = 335740994
	Isc_gfix_trn_not_valid = 335740995
	Isc_gfix_unexp_eoi = 335741012
	Isc_gfix_recon_fail = 335741018
	Isc_gfix_trn_unknown = 335741036
	Isc_gfix_mode_req = 335741038
	Isc_gfix_pzval_req = 335741042
	Isc_gfix_overval_req = 335741066
	Isc_dsql_dbkey_from_non_table = 336003074
	Isc_dsql_transitional_numeric = 336003075
	Isc_dsql_dialect_warning_expr = 336003076
	Isc_sql_db_dialect_dtype_unsupport = 336003077
	Isc_sql_dialect_conflict_num = 336003079
	Isc_dsql_warning_number_ambiguous = 336003080
	Isc_dsql_warning_number_ambiguous1 = 336003081
	Isc_dsql_warn_precision_ambiguous = 336003082
	Isc_dsql_warn_precision_ambiguous1 = 336003083
	Isc_dsql_warn_precision_ambiguous2 = 336003084
	Isc_dsql_ambiguous_field_name = 336003085
	Isc_dsql_udf_return_pos_err = 336003086
	Isc_dsql_invalid_label = 336003087
	Isc_dsql_datatypes_not_comparable = 336003088
	Isc_dsql_cursor_invalid = 336003089
	Isc_dsql_cursor_redefined = 336003090
	Isc_dsql_cursor_not_found = 336003091
	Isc_dsql_cursor_exists = 336003092
	Isc_dsql_cursor_rel_ambiguous = 336003093
	Isc_dsql_cursor_rel_not_found = 336003094
	Isc_dsql_cursor_not_open = 336003095
	Isc_dsql_type_not_supp_ext_tab = 336003096
	Isc_dsql_feature_not_supported_ods = 336003097
	Isc_primary_key_required = 336003098
	Isc_upd_ins_doesnt_match_pk = 336003099
	Isc_upd_ins_doesnt_match_matching = 336003100
	Isc_upd_ins_with_complex_view = 336003101
	Isc_dsql_incompatible_trigger_type = 336003102
	Isc_dsql_db_trigger_type_cant_change = 336003103
	Isc_dsql_record_version_table = 336003104
	Isc_dsql_invalid_sqlda_version = 336003105
	Isc_dsql_sqlvar_index = 336003106
	Isc_dsql_no_sqlind = 336003107
	Isc_dsql_no_sqldata = 336003108
	Isc_dsql_no_input_sqlda = 336003109
	Isc_dsql_no_output_sqlda = 336003110
	Isc_dsql_wrong_param_num = 336003111
	Isc_dsql_invalid_drop_ss_clause = 336003112
	Isc_upd_ins_cannot_default = 336003113
	Isc_dyn_filter_not_found = 336068645
	Isc_dyn_func_not_found = 336068649
	Isc_dyn_index_not_found = 336068656
	Isc_dyn_view_not_found = 336068662
	Isc_dyn_domain_not_found = 336068697
	Isc_dyn_cant_modify_auto_trig = 336068717
	Isc_dyn_dup_table = 336068740
	Isc_dyn_proc_not_found = 336068748
	Isc_dyn_exception_not_found = 336068752
	Isc_dyn_proc_param_not_found = 336068754
	Isc_dyn_trig_not_found = 336068755
	Isc_dyn_charset_not_found = 336068759
	Isc_dyn_collation_not_found = 336068760
	Isc_dyn_role_not_found = 336068763
	Isc_dyn_name_longer = 336068767
	Isc_dyn_column_does_not_exist = 336068784
	Isc_dyn_role_does_not_exist = 336068796
	Isc_dyn_no_grant_admin_opt = 336068797
	Isc_dyn_user_not_role_member = 336068798
	Isc_dyn_delete_role_failed = 336068799
	Isc_dyn_grant_role_to_user = 336068800
	Isc_dyn_inv_sql_role_name = 336068801
	Isc_dyn_dup_sql_role = 336068802
	Isc_dyn_kywd_spec_for_role = 336068803
	Isc_dyn_roles_not_supported = 336068804
	Isc_dyn_domain_name_exists = 336068812
	Isc_dyn_field_name_exists = 336068813
	Isc_dyn_dependency_exists = 336068814
	Isc_dyn_dtype_invalid = 336068815
	Isc_dyn_char_fld_too_small = 336068816
	Isc_dyn_invalid_dtype_conversion = 336068817
	Isc_dyn_dtype_conv_invalid = 336068818
	Isc_dyn_zero_len_id = 336068820
	Isc_dyn_gen_not_found = 336068822
	Isc_max_coll_per_charset = 336068829
	Isc_invalid_coll_attr = 336068830
	Isc_dyn_wrong_gtt_scope = 336068840
	Isc_dyn_coll_used_table = 336068843
	Isc_dyn_coll_used_domain = 336068844
	Isc_dyn_cannot_del_syscoll = 336068845
	Isc_dyn_cannot_del_def_coll = 336068846
	Isc_dyn_table_not_found = 336068849
	Isc_dyn_coll_used_procedure = 336068851
	Isc_dyn_scale_too_big = 336068852
	Isc_dyn_precision_too_small = 336068853
	Isc_dyn_miss_priv_warning = 336068855
	Isc_dyn_ods_not_supp_feature = 336068856
	Isc_dyn_cannot_addrem_computed = 336068857
	Isc_dyn_no_empty_pw = 336068858
	Isc_dyn_dup_index = 336068859
	Isc_dyn_package_not_found = 336068864
	Isc_dyn_schema_not_found = 336068865
	Isc_dyn_cannot_mod_sysproc = 336068866
	Isc_dyn_cannot_mod_systrig = 336068867
	Isc_dyn_cannot_mod_sysfunc = 336068868
	Isc_dyn_invalid_ddl_proc = 336068869
	Isc_dyn_invalid_ddl_trig = 336068870
	Isc_dyn_funcnotdef_package = 336068871
	Isc_dyn_procnotdef_package = 336068872
	Isc_dyn_funcsignat_package = 336068873
	Isc_dyn_procsignat_package = 336068874
	Isc_dyn_defvaldecl_package_proc = 336068875
	Isc_dyn_package_body_exists = 336068877
	Isc_dyn_invalid_ddl_func = 336068878
	Isc_dyn_newfc_oldsyntax = 336068879
	Isc_dyn_func_param_not_found = 336068886
	Isc_dyn_routine_param_not_found = 336068887
	Isc_dyn_routine_param_ambiguous = 336068888
	Isc_dyn_coll_used_function = 336068889
	Isc_dyn_domain_used_function = 336068890
	Isc_dyn_alter_user_no_clause = 336068891
	Isc_dyn_duplicate_package_item = 336068894
	Isc_dyn_cant_modify_sysobj = 336068895
	Isc_dyn_cant_use_zero_increment = 336068896
	Isc_dyn_cant_use_in_foreignkey = 336068897
	Isc_dyn_defvaldecl_package_func = 336068898
	Isc_dyn_cyclic_role = 336068900
	Isc_dyn_cant_use_zero_inc_ident = 336068904
	Isc_dyn_no_ddl_grant_opt_priv = 336068907
	Isc_dyn_no_grant_opt_priv = 336068908
	Isc_dyn_func_not_exist = 336068909
	Isc_dyn_proc_not_exist = 336068910
	Isc_dyn_pack_not_exist = 336068911
	Isc_dyn_trig_not_exist = 336068912
	Isc_dyn_view_not_exist = 336068913
	Isc_dyn_rel_not_exist = 336068914
	Isc_dyn_exc_not_exist = 336068915
	Isc_dyn_gen_not_exist = 336068916
	Isc_dyn_fld_not_exist = 336068917
	Isc_dyn_index_schema_must_match_table = 336068922
	Isc_dyn_trig_schema_must_match_table = 336068923
	Isc_dyn_cannot_mod_system_schema = 336068925
	Isc_dyn_cannot_drop_non_emptyschema = 336068926
	Isc_dyn_cannot_mod_obj_sys_schema = 336068927
	Isc_dyn_cannot_create_reserved_schema = 336068928
	Isc_dyn_cannot_infer_schema = 336068929
	Isc_dyn_dup_domain_rdb = 336069609
	Isc_dyn_dup_trigger_rdb = 336069610
	Isc_dyn_dup_package_rdb = 336069611
	Isc_dyn_dup_filter = 336069612
	Isc_dyn_job_not_found = 336069613
	Isc_dyn_invalid_ddl_job = 336069614
	Isc_dyn_alter_job_no_clause = 336069615
	Isc_dyn_ts_not_found = 336069616
	Isc_dyn_cant_set_ts_table = 336069617
	Isc_dyn_cant_modify_collation = 336069618
	Isc_dyn_cant_set_ts_index = 336069619
	Isc_dyn_foreign_server_not_found = 336069620
	Isc_dyn_user_mapping_not_found = 336069621
	Isc_dyn_job_already_exists = 336069622
	Isc_dyn_ts_already_exists = 336069623
	Isc_dyn_foreign_server_already_exists = 336069624
	Isc_dyn_cant_set_ts_external_table = 336069625
	Isc_gbak_unknown_switch = 336330753
	Isc_gbak_page_size_missing = 336330754
	Isc_gbak_page_size_toobig = 336330755
	Isc_gbak_redir_ouput_missing = 336330756
	Isc_gbak_switches_conflict = 336330757
	Isc_gbak_unknown_device = 336330758
	Isc_gbak_no_protection = 336330759
	Isc_gbak_page_size_not_allowed = 336330760
	Isc_gbak_multi_source_dest = 336330761
	Isc_gbak_filename_missing = 336330762
	Isc_gbak_dup_inout_names = 336330763
	Isc_gbak_inv_page_size = 336330764
	Isc_gbak_db_specified = 336330765
	Isc_gbak_db_exists = 336330766
	Isc_gbak_unk_device = 336330767
	Isc_gbak_blob_info_failed = 336330772
	Isc_gbak_unk_blob_item = 336330773
	Isc_gbak_get_seg_failed = 336330774
	Isc_gbak_close_blob_failed = 336330775
	Isc_gbak_open_blob_failed = 336330776
	Isc_gbak_put_blr_gen_id_failed = 336330777
	Isc_gbak_unk_type = 336330778
	Isc_gbak_comp_req_failed = 336330779
	Isc_gbak_start_req_failed = 336330780
	Isc_gbak_rec_failed = 336330781
	Isc_gbak_rel_req_failed = 336330782
	Isc_gbak_db_info_failed = 336330783
	Isc_gbak_no_db_desc = 336330784
	Isc_gbak_db_create_failed = 336330785
	Isc_gbak_decomp_len_error = 336330786
	Isc_gbak_tbl_missing = 336330787
	Isc_gbak_blob_col_missing = 336330788
	Isc_gbak_create_blob_failed = 336330789
	Isc_gbak_put_seg_failed = 336330790
	Isc_gbak_rec_len_exp = 336330791
	Isc_gbak_inv_rec_len = 336330792
	Isc_gbak_exp_data_type = 336330793
	Isc_gbak_gen_id_failed = 336330794
	Isc_gbak_unk_rec_type = 336330795
	Isc_gbak_inv_bkup_ver = 336330796
	Isc_gbak_missing_bkup_desc = 336330797
	Isc_gbak_string_trunc = 336330798
	Isc_gbak_cant_rest_record = 336330799
	Isc_gbak_send_failed = 336330800
	Isc_gbak_no_tbl_name = 336330801
	Isc_gbak_unexp_eof = 336330802
	Isc_gbak_db_format_too_old = 336330803
	Isc_gbak_inv_array_dim = 336330804
	Isc_gbak_xdr_len_expected = 336330807
	Isc_gbak_open_bkup_error = 336330817
	Isc_gbak_open_error = 336330818
	Isc_gbak_missing_block_fac = 336330934
	Isc_gbak_inv_block_fac = 336330935
	Isc_gbak_block_fac_specified = 336330936
	Isc_gbak_missing_username = 336330940
	Isc_gbak_missing_password = 336330941
	Isc_gbak_missing_skipped_bytes = 336330952
	Isc_gbak_inv_skipped_bytes = 336330953
	Isc_gbak_err_restore_charset = 336330965
	Isc_gbak_err_restore_collation = 336330967
	Isc_gbak_read_error = 336330972
	Isc_gbak_write_error = 336330973
	Isc_gbak_db_in_use = 336330985
	Isc_gbak_sysmemex = 336330990
	Isc_gbak_restore_role_failed = 336331002
	Isc_gbak_role_op_missing = 336331005
	Isc_gbak_page_buffers_missing = 336331010
	Isc_gbak_page_buffers_wrong_param = 336331011
	Isc_gbak_page_buffers_restore = 336331012
	Isc_gbak_inv_size = 336331014
	Isc_gbak_file_outof_sequence = 336331015
	Isc_gbak_join_file_missing = 336331016
	Isc_gbak_stdin_not_supptd = 336331017
	Isc_gbak_stdout_not_supptd = 336331018
	Isc_gbak_bkup_corrupt = 336331019
	Isc_gbak_unk_db_file_spec = 336331020
	Isc_gbak_hdr_write_failed = 336331021
	Isc_gbak_disk_space_ex = 336331022
	Isc_gbak_size_lt_min = 336331023
	Isc_gbak_svc_name_missing = 336331025
	Isc_gbak_not_ownr = 336331026
	Isc_gbak_mode_req = 336331031
	Isc_gbak_just_data = 336331033
	Isc_gbak_data_only = 336331034
	Isc_gbak_missing_interval = 336331078
	Isc_gbak_wrong_interval = 336331079
	Isc_gbak_verify_verbint = 336331081
	Isc_gbak_option_only_restore = 336331082
	Isc_gbak_option_only_backup = 336331083
	Isc_gbak_option_conflict = 336331084
	Isc_gbak_param_conflict = 336331085
	Isc_gbak_option_repeated = 336331086
	Isc_gbak_max_dbkey_recursion = 336331091
	Isc_gbak_max_dbkey_length = 336331092
	Isc_gbak_invalid_metadata = 336331093
	Isc_gbak_invalid_data = 336331094
	Isc_gbak_inv_bkup_ver2 = 336331096
	Isc_gbak_db_format_too_old2 = 336331100
	Isc_gbak_db_not_online = 336331762
	Isc_gbak_gfix_online = 336331763
	Isc_gbak_db_not_online_reason = 336331778
	Isc_dsql_too_old_ods = 336397205
	Isc_dsql_table_not_found = 336397206
	Isc_dsql_view_not_found = 336397207
	Isc_dsql_line_col_error = 336397208
	Isc_dsql_unknown_pos = 336397209
	Isc_dsql_no_dup_name = 336397210
	Isc_dsql_too_many_values = 336397211
	Isc_dsql_no_array_computed = 336397212
	Isc_dsql_implicit_domain_name = 336397213
	Isc_dsql_only_can_subscript_array = 336397214
	Isc_dsql_max_sort_items = 336397215
	Isc_dsql_max_group_items = 336397216
	Isc_dsql_conflicting_sort_field = 336397217
	Isc_dsql_derived_table_more_columns = 336397218
	Isc_dsql_derived_table_less_columns = 336397219
	Isc_dsql_derived_field_unnamed = 336397220
	Isc_dsql_derived_field_dup_name = 336397221
	Isc_dsql_derived_alias_select = 336397222
	Isc_dsql_derived_alias_field = 336397223
	Isc_dsql_auto_field_bad_pos = 336397224
	Isc_dsql_cte_wrong_reference = 336397225
	Isc_dsql_cte_cycle = 336397226
	Isc_dsql_cte_outer_join = 336397227
	Isc_dsql_cte_mult_references = 336397228
	Isc_dsql_cte_not_a_union = 336397229
	Isc_dsql_cte_nonrecurs_after_recurs = 336397230
	Isc_dsql_cte_wrong_clause = 336397231
	Isc_dsql_cte_union_all = 336397232
	Isc_dsql_cte_miss_nonrecursive = 336397233
	Isc_dsql_cte_nested_with = 336397234
	Isc_dsql_col_more_than_once_using = 336397235
	Isc_dsql_unsupp_feature_dialect = 336397236
	Isc_dsql_cte_not_used = 336397237
	Isc_dsql_col_more_than_once_view = 336397238
	Isc_dsql_unsupported_in_auto_trans = 336397239
	Isc_dsql_eval_unknode = 336397240
	Isc_dsql_agg_wrongarg = 336397241
	Isc_dsql_agg2_wrongarg = 336397242
	Isc_dsql_nodateortime_pm_string = 336397243
	Isc_dsql_invalid_datetime_subtract = 336397244
	Isc_dsql_invalid_dateortime_add = 336397245
	Isc_dsql_invalid_type_minus_date = 336397246
	Isc_dsql_nostring_addsub_dial3 = 336397247
	Isc_dsql_invalid_type_addsub_dial3 = 336397248
	Isc_dsql_invalid_type_multip_dial1 = 336397249
	Isc_dsql_nostring_multip_dial3 = 336397250
	Isc_dsql_invalid_type_multip_dial3 = 336397251
	Isc_dsql_mustuse_numeric_div_dial1 = 336397252
	Isc_dsql_nostring_div_dial3 = 336397253
	Isc_dsql_invalid_type_div_dial3 = 336397254
	Isc_dsql_nostring_neg_dial3 = 336397255
	Isc_dsql_invalid_type_neg = 336397256
	Isc_dsql_max_distinct_items = 336397257
	Isc_dsql_alter_charset_failed = 336397258
	Isc_dsql_comment_on_failed = 336397259
	Isc_dsql_create_func_failed = 336397260
	Isc_dsql_alter_func_failed = 336397261
	Isc_dsql_create_alter_func_failed = 336397262
	Isc_dsql_drop_func_failed = 336397263
	Isc_dsql_recreate_func_failed = 336397264
	Isc_dsql_create_proc_failed = 336397265
	Isc_dsql_alter_proc_failed = 336397266
	Isc_dsql_create_alter_proc_failed = 336397267
	Isc_dsql_drop_proc_failed = 336397268
	Isc_dsql_recreate_proc_failed = 336397269
	Isc_dsql_create_trigger_failed = 336397270
	Isc_dsql_alter_trigger_failed = 336397271
	Isc_dsql_create_alter_trigger_failed = 336397272
	Isc_dsql_drop_trigger_failed = 336397273
	Isc_dsql_recreate_trigger_failed = 336397274
	Isc_dsql_create_collation_failed = 336397275
	Isc_dsql_drop_collation_failed = 336397276
	Isc_dsql_create_domain_failed = 336397277
	Isc_dsql_alter_domain_failed = 336397278
	Isc_dsql_drop_domain_failed = 336397279
	Isc_dsql_create_except_failed = 336397280
	Isc_dsql_alter_except_failed = 336397281
	Isc_dsql_create_alter_except_failed = 336397282
	Isc_dsql_recreate_except_failed = 336397283
	Isc_dsql_drop_except_failed = 336397284
	Isc_dsql_create_sequence_failed = 336397285
	Isc_dsql_create_table_failed = 336397286
	Isc_dsql_alter_table_failed = 336397287
	Isc_dsql_drop_table_failed = 336397288
	Isc_dsql_recreate_table_failed = 336397289
	Isc_dsql_create_pack_failed = 336397290
	Isc_dsql_alter_pack_failed = 336397291
	Isc_dsql_create_alter_pack_failed = 336397292
	Isc_dsql_drop_pack_failed = 336397293
	Isc_dsql_recreate_pack_failed = 336397294
	Isc_dsql_create_pack_body_failed = 336397295
	Isc_dsql_drop_pack_body_failed = 336397296
	Isc_dsql_recreate_pack_body_failed = 336397297
	Isc_dsql_create_view_failed = 336397298
	Isc_dsql_alter_view_failed = 336397299
	Isc_dsql_create_alter_view_failed = 336397300
	Isc_dsql_recreate_view_failed = 336397301
	Isc_dsql_drop_view_failed = 336397302
	Isc_dsql_drop_sequence_failed = 336397303
	Isc_dsql_recreate_sequence_failed = 336397304
	Isc_dsql_drop_index_failed = 336397305
	Isc_dsql_drop_filter_failed = 336397306
	Isc_dsql_drop_shadow_failed = 336397307
	Isc_dsql_drop_role_failed = 336397308
	Isc_dsql_drop_user_failed = 336397309
	Isc_dsql_create_role_failed = 336397310
	Isc_dsql_alter_role_failed = 336397311
	Isc_dsql_alter_index_failed = 336397312
	Isc_dsql_alter_database_failed = 336397313
	Isc_dsql_create_shadow_failed = 336397314
	Isc_dsql_create_filter_failed = 336397315
	Isc_dsql_create_index_failed = 336397316
	Isc_dsql_create_user_failed = 336397317
	Isc_dsql_alter_user_failed = 336397318
	Isc_dsql_grant_failed = 336397319
	Isc_dsql_revoke_failed = 336397320
	Isc_dsql_cte_recursive_aggregate = 336397321
	Isc_dsql_mapping_failed = 336397322
	Isc_dsql_alter_sequence_failed = 336397323
	Isc_dsql_create_generator_failed = 336397324
	Isc_dsql_set_generator_failed = 336397325
	Isc_dsql_wlock_simple = 336397326
	Isc_dsql_firstskip_rows = 336397327
	Isc_dsql_wlock_aggregates = 336397328
	Isc_dsql_wlock_conflict = 336397329
	Isc_dsql_max_exception_arguments = 336397330
	Isc_dsql_string_byte_length = 336397331
	Isc_dsql_string_char_length = 336397332
	Isc_dsql_max_nesting = 336397333
	Isc_dsql_recreate_user_failed = 336397334
	Isc_dsql_table_value_many_columns = 336397335
	Isc_dsql_create_schema_failed = 336397336
	Isc_dsql_drop_schema_failed = 336397337
	Isc_dsql_recreate_schema_failed = 336397338
	Isc_dsql_alter_schema_failed = 336397339
	Isc_dsql_create_alter_schema_failed = 336397340
	Isc_dsql_create_policy_failed = 336398288
	Isc_dsql_alter_policy_failed = 336398289
	Isc_dsql_drop_policy_failed = 336398290
	Isc_dsql_agg_wrongtype = 336398291
	Isc_dsql_create_job_failed = 336398292
	Isc_dsql_alter_job_failed = 336398293
	Isc_dsql_create_alter_job_failed = 336398294
	Isc_dsql_drop_job_failed = 336398295
	Isc_dsql_recreate_job_failed = 336398296
	Isc_dsql_create_ts_failed = 336398297
	Isc_dsql_alter_ts_failed = 336398298
	Isc_dsql_create_alter_ts_failed = 336398299
	Isc_dsql_drop_ts_failed = 336398300
	Isc_dsql_recreate_ts_failed = 336398301
	Isc_dsql_create_foreign_server_failed = 336398302
	Isc_dsql_alter_foreign_server_failed = 336398303
	Isc_dsql_create_alter_foreign_server_failed = 336398304
	Isc_dsql_recreate_foreign_server_failed = 336398305
	Isc_dsql_drop_foreign_server_failed = 336398306
	Isc_dsql_create_user_mapping_failed = 336398307
	Isc_dsql_alter_user_mapping_failed = 336398308
	Isc_dsql_create_alter_user_mapping_failed = 336398309
	Isc_dsql_recreate_user_mapping_failed = 336398310
	Isc_dsql_drop_user_mapping_failed = 336398311
	Isc_dsql_foreign_table_not_found = 336398312
	Isc_dsql_create_foreign_table_failed = 336398313
	Isc_dsql_alter_foreign_table_failed = 336398314
	Isc_dsql_drop_foreign_table_failed = 336398315
	Isc_dsql_recreate_foreign_table_failed = 336398316
	Isc_gsec_cant_open_db = 336723983
	Isc_gsec_switches_error = 336723984
	Isc_gsec_no_op_spec = 336723985
	Isc_gsec_no_usr_name = 336723986
	Isc_gsec_err_add = 336723987
	Isc_gsec_err_modify = 336723988
	Isc_gsec_err_find_mod = 336723989
	Isc_gsec_err_rec_not_found = 336723990
	Isc_gsec_err_delete = 336723991
	Isc_gsec_err_find_del = 336723992
	Isc_gsec_err_find_disp = 336723996
	Isc_gsec_inv_param = 336723997
	Isc_gsec_op_specified = 336723998
	Isc_gsec_pw_specified = 336723999
	Isc_gsec_uid_specified = 336724000
	Isc_gsec_gid_specified = 336724001
	Isc_gsec_proj_specified = 336724002
	Isc_gsec_org_specified = 336724003
	Isc_gsec_fname_specified = 336724004
	Isc_gsec_mname_specified = 336724005
	Isc_gsec_lname_specified = 336724006
	Isc_gsec_inv_switch = 336724008
	Isc_gsec_amb_switch = 336724009
	Isc_gsec_no_op_specified = 336724010
	Isc_gsec_params_not_allowed = 336724011
	Isc_gsec_incompat_switch = 336724012
	Isc_gsec_inv_username = 336724044
	Isc_gsec_inv_pw_length = 336724045
	Isc_gsec_db_specified = 336724046
	Isc_gsec_db_admin_specified = 336724047
	Isc_gsec_db_admin_pw_specified = 336724048
	Isc_gsec_sql_role_specified = 336724049
	Isc_gstat_unknown_switch = 336920577
	Isc_gstat_retry = 336920578
	Isc_gstat_wrong_ods = 336920579
	Isc_gstat_unexpected_eof = 336920580
	Isc_gstat_open_err = 336920605
	Isc_gstat_read_err = 336920606
	Isc_gstat_sysmemex = 336920607
	Isc_fbsvcmgr_bad_am = 336986113
	Isc_fbsvcmgr_bad_wm = 336986114
	Isc_fbsvcmgr_bad_rs = 336986115
	Isc_fbsvcmgr_info_err = 336986116
	Isc_fbsvcmgr_query_err = 336986117
	Isc_fbsvcmgr_switch_unknown = 336986118
	Isc_fbsvcmgr_bad_sm = 336986159
	Isc_fbsvcmgr_fp_open = 336986160
	Isc_fbsvcmgr_fp_read = 336986161
	Isc_fbsvcmgr_fp_empty = 336986162
	Isc_fbsvcmgr_bad_arg = 336986164
	Isc_fbsvcmgr_info_limbo = 336986170
	Isc_fbsvcmgr_limbo_state = 336986171
	Isc_fbsvcmgr_limbo_advise = 336986172
	Isc_fbsvcmgr_bad_rm = 336986173
	Isc_utl_trusted_switch = 337051649
	Isc_nbackup_missing_param = 337117213
	Isc_nbackup_allowed_switches = 337117214
	Isc_nbackup_unknown_param = 337117215
	Isc_nbackup_unknown_switch = 337117216
	Isc_nbackup_nofetchpw_svc = 337117217
	Isc_nbackup_pwfile_error = 337117218
	Isc_nbackup_size_with_lock = 337117219
	Isc_nbackup_no_switch = 337117220
	Isc_nbackup_err_read = 337117223
	Isc_nbackup_err_write = 337117224
	Isc_nbackup_err_seek = 337117225
	Isc_nbackup_err_opendb = 337117226
	Isc_nbackup_err_fadvice = 337117227
	Isc_nbackup_err_createdb = 337117228
	Isc_nbackup_err_openbk = 337117229
	Isc_nbackup_err_createbk = 337117230
	Isc_nbackup_err_eofdb = 337117231
	Isc_nbackup_fixup_wrongstate = 337117232
	Isc_nbackup_err_db = 337117233
	Isc_nbackup_userpw_toolong = 337117234
	Isc_nbackup_lostrec_db = 337117235
	Isc_nbackup_lostguid_db = 337117236
	Isc_nbackup_err_eofhdrdb = 337117237
	Isc_nbackup_db_notlock = 337117238
	Isc_nbackup_lostguid_bk = 337117239
	Isc_nbackup_page_changed = 337117240
	Isc_nbackup_dbsize_inconsistent = 337117241
	Isc_nbackup_failed_lzbk = 337117242
	Isc_nbackup_err_eofhdrbk = 337117243
	Isc_nbackup_invalid_incbk = 337117244
	Isc_nbackup_unsupvers_incbk = 337117245
	Isc_nbackup_invlevel_incbk = 337117246
	Isc_nbackup_wrong_orderbk = 337117247
	Isc_nbackup_err_eofbk = 337117248
	Isc_nbackup_err_copy = 337117249
	Isc_nbackup_err_eofhdr_restdb = 337117250
	Isc_nbackup_lostguid_l0bk = 337117251
	Isc_nbackup_switchd_parameter = 337117255
	Isc_nbackup_user_stop = 337117257
	Isc_nbackup_deco_parse = 337117259
	Isc_nbackup_lostrec_guid_db = 337117261
	Isc_nbackup_seq_misuse = 337117265
	Isc_nbackup_wrong_param = 337117268
	Isc_nbackup_clean_hist_misuse = 337117269
	Isc_nbackup_clean_hist_missed = 337117270
	Isc_nbackup_keep_hist_missed = 337117271
	Isc_nbackup_second_keep_switch = 337117272
	Isc_trace_conflict_acts = 337182750
	Isc_trace_act_notfound = 337182751
	Isc_trace_switch_once = 337182752
	Isc_trace_param_val_miss = 337182753
	Isc_trace_param_invalid = 337182754
	Isc_trace_switch_unknown = 337182755
	Isc_trace_switch_svc_only = 337182756
	Isc_trace_switch_user_only = 337182757
	Isc_trace_switch_param_miss = 337182758
	Isc_trace_param_act_notcompat = 337182759
	Isc_trace_mandatory_switch_miss = 337182760
	Isc_diff_incmp_sw = 337510415
	Isc_diff_unicode_init_error = 337510416
	Isc_diff_replconf_section_not_found = 337510417
	Isc_diff_param_val_miss = 337510437
)

var errmsgs = map[int]fberror{
	Isc_arith_except: {"arithmetic exception, numeric overflow, or string truncation\n", "22000"},
	Isc_bad_dbkey: {"invalid database key\n", "HY000"},
	Isc_bad_db_format: {"file @1 is not a valid database\n", "HY000"},
	Isc_bad_db_handle: {"invalid database handle (no active connection)\n", "08003"},
	Isc_bad_dpb_content: {"bad parameters on attach or create database\n", "HY000"},
	Isc_bad_dpb_form: {"unrecognized database parameter block\n", "HY000"},
	Isc_bad_req_handle: {"invalid request handle\n", "42000"},
	Isc_bad_segstr_handle: {"invalid BLOB handle\n", "42000"},
	Isc_bad_segstr_id: {"invalid BLOB ID\n", "42000"},
	Isc_bad_tpb_content: {"invalid parameter in transaction parameter block\n", "HY000"},
	Isc_bad_tpb_form: {"invalid format for transaction parameter block\n", "HY000"},
	Isc_bad_trans_handle: {"invalid transaction handle (expecting explicit transaction start)\n", "HY000"},
	Isc_bug_check: {"internal Red Database consistency check (@1)\n", "XX000"},
	Isc_convert_error: {"conversion error from string \"@1\"\n", "22018"},
	Isc_db_corrupt: {"database file appears corrupt (@1)\n", "XX001"},
	Isc_deadlock: {"deadlock\n", "40001"},
	Isc_excess_trans: {"attempt to start more than @1 transactions\n", "HY000"},
	Isc_from_no_match: {"no match for first value expression\n", "01000"},
	Isc_infinap: {"information type inappropriate for object specified\n", "HY000"},
	Isc_infona: {"no information of this type available for object specified\n", "HY000"},
	Isc_infunk: {"unknown information item\n", "HY000"},
	Isc_integ_fail: {"action cancelled by trigger (@1) to preserve data integrity\n", "27000"},
	Isc_invalid_blr: {"invalid request BLR at offset @1\n", "HY000"},
	Isc_io_error: {"I/O error during \"@1\" operation for file \"@2\"\n", "08001"},
	Isc_lock_conflict: {"lock conflict on no wait transaction\n", "40001"},
	Isc_metadata_corrupt: {"corrupt system table\n", "XX001"},
	Isc_not_valid: {"validation error for column @1, value \"@2\"\n", "23000"},
	Isc_no_cur_rec: {"no current record for fetch operation\n", "22000"},
	Isc_no_dup: {"attempt to store duplicate value (visible to active transactions) in unique index @1\n", "23000"},
	Isc_no_finish: {"program attempted to exit without finishing database\n", "HY000"},
	Isc_no_meta_update: {"unsuccessful metadata update\n", "42000"},
	Isc_no_priv: {"no permission for @1 access to @2 @3\n", "28000"},
	Isc_no_recon: {"transaction is not in limbo\n", "HY000"},
	Isc_no_record: {"invalid database key\n", "01000"},
	Isc_no_segstr_close: {"BLOB was not closed\n", "HY000"},
	Isc_obsolete_metadata: {"metadata is obsolete\n", "HY000"},
	Isc_open_trans: {"cannot disconnect database with open transactions (@1 active)\n", "01002"},
	Isc_port_len: {"message length error (encountered @1, expected @2)\n", "HY000"},
	Isc_read_only_field: {"attempted update of read-only column @1\n", "42000"},
	Isc_read_only_rel: {"attempted update of read-only table\n", "42000"},
	Isc_read_only_trans: {"attempted update during read-only transaction\n", "25006"},
	Isc_read_only_view: {"cannot update read-only view @1\n", "42000"},
	Isc_req_no_trans: {"no transaction for request\n", "25000"},
	Isc_req_sync: {"request synchronization error\n", "HY000"},
	Isc_req_wrong_db: {"request referenced an unavailable database\n", "HY000"},
	Isc_segment: {"segment buffer length shorter than expected\n", "01000"},
	Isc_segstr_eof: {"attempted retrieval of more segments than exist\n", "01000"},
	Isc_segstr_no_op: {"attempted invalid operation on a BLOB\n", "HY000"},
	Isc_segstr_no_read: {"attempted read of a new, open BLOB\n", "HY000"},
	Isc_segstr_no_trans: {"attempted action on BLOB outside transaction\n", "HY000"},
	Isc_segstr_no_write: {"attempted write to read-only BLOB\n", "HY000"},
	Isc_segstr_wrong_db: {"attempted reference to BLOB in unavailable database\n", "HY000"},
	Isc_sys_request: {"operating system directive @1 failed\n", "HY000"},
	Isc_stream_eof: {"attempt to fetch past the last record in a record stream\n", "HY109"},
	Isc_unavailable: {"unavailable database\n", "08001"},
	Isc_unres_rel: {"table @1 was omitted from the transaction reserving list\n", "HY000"},
	Isc_uns_ext: {"request includes a DSRI extension not supported in this implementation\n", "HY000"},
	Isc_wish_list: {"feature is not supported\n", "0A000"},
	Isc_wrong_ods: {"unsupported on-disk structure for file @1; found @2.@3, support @4.@5\n", "HY000"},
	Isc_wronumarg: {"wrong number of arguments on call\n", "07002"},
	Isc_imp_exc: {"Implementation limit exceeded\n", "54000"},
	Isc_random: {"@1\n", "HY000"},
	Isc_fatal_conflict: {"unrecoverable conflict with limbo transaction @1\n", "HY000"},
	Isc_badblk: {"internal error\n", "XX000"},
	Isc_invpoolcl: {"internal error\n", "XX000"},
	Isc_nopoolids: {"too many requests\n", "HY000"},
	Isc_relbadblk: {"internal error\n", "XX000"},
	Isc_blktoobig: {"block size exceeds implementation restriction\n", "54000"},
	Isc_bufexh: {"buffer exhausted\n", "HY001"},
	Isc_syntaxerr: {"BLR syntax error: expected @1 at offset @2, encountered @3\n", "HY000"},
	Isc_bufinuse: {"buffer in use\n", "HY000"},
	Isc_bdbincon: {"internal error\n", "XX000"},
	Isc_reqinuse: {"request in use\n", "HY000"},
	Isc_badodsver: {"incompatible version of on-disk structure\n", "HY000"},
	Isc_relnotdef: {"table @1 is not defined\n", "42S02"},
	Isc_fldnotdef: {"column @1 is not defined in table @2\n", "42S22"},
	Isc_dirtypage: {"internal error\n", "XX000"},
	Isc_waifortra: {"internal error\n", "XX000"},
	Isc_doubleloc: {"internal error\n", "XX000"},
	Isc_nodnotfnd: {"internal error\n", "XX000"},
	Isc_dupnodfnd: {"internal error\n", "XX000"},
	Isc_locnotmar: {"internal error\n", "XX000"},
	Isc_badpagtyp: {"page @1 is of wrong type (expected @2, found @3)\n", "42000"},
	Isc_corrupt: {"database corrupted\n", "XX000"},
	Isc_badpage: {"checksum error on database page @1\n", "42000"},
	Isc_badindex: {"index is broken\n", "XX002"},
	Isc_dbbnotzer: {"database handle not zero\n", "HY000"},
	Isc_tranotzer: {"transaction handle not zero\n", "25000"},
	Isc_trareqmis: {"transaction--request mismatch (synchronization error)\n", "25000"},
	Isc_badhndcnt: {"bad handle count\n", "HY000"},
	Isc_wrotpbver: {"wrong version of transaction parameter block\n", "HY000"},
	Isc_wroblrver: {"unsupported BLR version (expected @1, encountered @2)\n", "HY000"},
	Isc_wrodpbver: {"wrong version of database parameter block\n", "HY000"},
	Isc_blobnotsup: {"BLOB and array data types are not supported for @1 operation\n", "42000"},
	Isc_badrelation: {"database corrupted\n", "XX001"},
	Isc_nodetach: {"internal error\n", "XX000"},
	Isc_notremote: {"internal error\n", "XX000"},
	Isc_trainlim: {"transaction in limbo\n", "25000"},
	Isc_notinlim: {"transaction not in limbo\n", "25000"},
	Isc_traoutsta: {"transaction outstanding\n", "25000"},
	Isc_connect_reject: {"connection rejected by remote interface\n", "08004"},
	Isc_dbfile: {"internal error\n", "XX000"},
	Isc_orphan: {"internal error\n", "XX000"},
	Isc_no_lock_mgr: {"no lock manager available\n", "HY000"},
	Isc_ctxinuse: {"context already in use (BLR error)\n", "HY000"},
	Isc_ctxnotdef: {"context not defined (BLR error)\n", "HY000"},
	Isc_datnotsup: {"data operation not supported\n", "42000"},
	Isc_badmsgnum: {"undefined message number\n", "HY000"},
	Isc_badparnum: {"undefined parameter number\n", "HY000"},
	Isc_virmemexh: {"unable to allocate memory from operating system\n", "HY001"},
	Isc_blocking_signal: {"blocking signal has been received\n", "HY000"},
	Isc_lockmanerr: {"lock manager error\n", "40001"},
	Isc_journerr: {"communication error with journal \"@1\"\n", "HY000"},
	Isc_keytoobig: {"key size exceeds implementation restriction for index @1\n", "42000"},
	Isc_nullsegkey: {"null segment of UNIQUE KEY\n", "HY000"},
	Isc_sqlerr: {"SQL error code = @1\n", "00000"},
	Isc_wrodynver: {"wrong DYN version\n", "HY000"},
	Isc_funnotdef: {"function @1 is not defined\n", "39000"},
	Isc_funmismat: {"function @1 could not be matched\n", "39000"},
	Isc_bad_msg_vec: {"\n", "22001"},
	Isc_bad_detach: {"database detach completed with errors\n", "08007"},
	Isc_noargacc_read: {"database system cannot read argument @1\n", "HY000"},
	Isc_noargacc_write: {"database system cannot write argument @1\n", "HY000"},
	Isc_read_only: {"operation not supported\n", "42000"},
	Isc_ext_err: {"@1 extension error\n", "42000"},
	Isc_non_updatable: {"not updatable\n", "42000"},
	Isc_no_rollback: {"no rollback performed\n", "25S02"},
	Isc_bad_sec_info: {"\n", "HY000"},
	Isc_invalid_sec_info: {"\n", "28000"},
	Isc_misc_interpreted: {"@1\n", "HY000"},
	Isc_update_conflict: {"update conflicts with concurrent update\n", "40001"},
	Isc_unlicensed: {"product @1 is not licensed\n", "HY000"},
	Isc_obj_in_use: {"object @1 is in use\n", "42000"},
	Isc_nofilter: {"filter not found to convert type @1 to type @2\n", "HY000"},
	Isc_shadow_accessed: {"cannot attach active shadow file\n", "00000"},
	Isc_invalid_sdl: {"invalid slice description language at offset @1\n", "HY000"},
	Isc_out_of_bounds: {"subscript out of bounds\n", "42000"},
	Isc_invalid_dimension: {"column not array or invalid dimensions (expected @1, encountered @2)\n", "42000"},
	Isc_rec_in_limbo: {"record from transaction @1 is stuck in limbo\n", "HY000"},
	Isc_shadow_missing: {"a file in manual shadow @1 is unavailable\n", "HY000"},
	Isc_cant_validate: {"secondary server attachments cannot validate databases\n", "HY000"},
	Isc_cant_start_journal: {"secondary server attachments cannot start journaling\n", "HY000"},
	Isc_gennotdef: {"generator @1 is not defined\n", "42000"},
	Isc_cant_start_logging: {"secondary server attachments cannot start logging\n", "HY000"},
	Isc_bad_segstr_type: {"invalid BLOB type for operation\n", "42000"},
	Isc_foreign_key: {"violation of FOREIGN KEY constraint @1 on table @2\n", "23000"},
	Isc_high_minor: {"minor version too high found @1 expected @2\n", "HY000"},
	Isc_tra_state: {"transaction @1 is @2\n", "00000"},
	Isc_trans_invalid: {"transaction marked invalid and cannot be committed\n", "25000"},
	Isc_buf_invalid: {"cache buffer for page @1 invalid\n", "HY000"},
	Isc_indexnotdefined: {"there is no index in table @1 with id @2\n", "42S12"},
	Isc_login: {"Your user name and password are not defined. Ask your database administrator to set up a Red Database login.\n", "28000"},
	Isc_invalid_bookmark: {"invalid bookmark handle\n", "HY111"},
	Isc_bad_lock_level: {"invalid lock level @1\n", "HY000"},
	Isc_relation_lock: {"lock on table @1 conflicts with existing lock\n", "40001"},
	Isc_record_lock: {"requested record lock conflicts with existing lock\n", "40001"},
	Isc_max_idx: {"maximum indexes per table (@1) exceeded\n", "54000"},
	Isc_jrn_enable: {"enable journal for database before starting online dump\n", "HY000"},
	Isc_old_failure: {"online dump failure. Retry dump\n", "HY000"},
	Isc_old_in_progress: {"an online dump is already in progress\n", "HY000"},
	Isc_old_no_space: {"no more disk/tape space.  Cannot continue online dump\n", "HY000"},
	Isc_no_wal_no_jrn: {"journaling allowed only if database has Write-ahead Log\n", "HY000"},
	Isc_num_old_files: {"maximum number of online dump files that can be specified is 16\n", "HY000"},
	Isc_wal_file_open: {"error in opening Write-ahead Log file during recovery\n", "HY000"},
	Isc_bad_stmt_handle: {"invalid statement handle\n", "26000"},
	Isc_wal_failure: {"Write-ahead log subsystem failure\n", "HY000"},
	Isc_walw_err: {"WAL Writer error\n", "HY000"},
	Isc_logh_small: {"Log file header of @1 too small\n", "HY000"},
	Isc_logh_inv_version: {"Invalid version of log file @1\n", "HY000"},
	Isc_logh_open_flag: {"Log file @1 not latest in the chain but open flag still set\n", "HY000"},
	Isc_logh_open_flag2: {"Log file @1 not closed properly; database recovery may be required\n", "HY000"},
	Isc_logh_diff_dbname: {"Database name in the log file @1 is different\n", "HY000"},
	Isc_logf_unexpected_eof: {"Unexpected end of log file @1 at offset @2\n", "HY000"},
	Isc_logr_incomplete: {"Incomplete log record at offset @1 in log file @2\n", "HY000"},
	Isc_logr_header_small: {"Log record header too small at offset @1 in log file @2\n", "HY000"},
	Isc_logb_small: {"Log block too small at offset @1 in log file @2\n", "HY000"},
	Isc_wal_illegal_attach: {"Illegal attempt to attach to an uninitialized WAL segment for @1\n", "HY000"},
	Isc_wal_invalid_wpb: {"Invalid WAL parameter block option @1\n", "HY000"},
	Isc_wal_err_rollover: {"Cannot roll over to the next log file @1\n", "HY000"},
	Isc_no_wal: {"database does not use Write-ahead Log\n", "HY000"},
	Isc_drop_wal: {"cannot drop log file when journaling is enabled\n", "HY000"},
	Isc_stream_not_defined: {"reference to invalid stream number\n", "HY000"},
	Isc_wal_subsys_error: {"WAL subsystem encountered error\n", "HY000"},
	Isc_wal_subsys_corrupt: {"WAL subsystem corrupted\n", "XX000"},
	Isc_no_archive: {"must specify archive file when enabling long term journal for databases with round-robin log files\n", "HY000"},
	Isc_shutinprog: {"database @1 shutdown in progress\n", "HY000"},
	Isc_range_in_use: {"refresh range number @1 already in use\n", "42000"},
	Isc_range_not_found: {"refresh range number @1 not found\n", "42000"},
	Isc_charset_not_found: {"CHARACTER SET @1 is not defined\n", "2C000"},
	Isc_lock_timeout: {"lock time-out on wait transaction\n", "40001"},
	Isc_prcnotdef: {"procedure @1 is not defined\n", "42000"},
	Isc_prcmismat: {"Parameter mismatch for procedure @1\n", "07001"},
	Isc_wal_bugcheck: {"Database @1: WAL subsystem bug for pid @2\n@3\n", "XX000"},
	Isc_wal_cant_expand: {"Could not expand the WAL segment for database @1\n", "HY000"},
	Isc_codnotdef: {"status code @1 unknown\n", "HY000"},
	Isc_xcpnotdef: {"exception @1 not defined\n", "HY000"},
	Isc_except: {"exception @1\n", "HY000"},
	Isc_cache_restart: {"restart shared cache manager\n", "HY013"},
	Isc_bad_lock_handle: {"invalid lock handle\n", "HY000"},
	Isc_jrn_present: {"long-term journaling already enabled\n", "HY000"},
	Isc_wal_err_rollover2: {"Unable to roll over please see Red Database log.\n", "HY000"},
	Isc_wal_err_logwrite: {"WAL I/O error.  Please see Red Database log.\n", "HY000"},
	Isc_wal_err_jrn_comm: {"WAL writer - Journal server communication error.  Please see Red Database log.\n", "HY000"},
	Isc_wal_err_expansion: {"WAL buffers cannot be increased.  Please see Red Database log.\n", "HY000"},
	Isc_wal_err_setup: {"WAL setup error.  Please see Red Database log.\n", "HY000"},
	Isc_wal_err_ww_sync: {"obsolete\n", "HY000"},
	Isc_wal_err_ww_start: {"Cannot start WAL writer for the database @1\n", "HY000"},
	Isc_shutdown: {"database @1 shutdown\n", "HY000"},
	Isc_existing_priv_mod: {"cannot modify an existing user privilege\n", "28000"},
	Isc_primary_key_ref: {"Cannot delete PRIMARY KEY being used in FOREIGN KEY definition.\n", "42000"},
	Isc_primary_key_notnull: {"Column used in a PRIMARY constraint must be NOT NULL.\n", "42000"},
	Isc_ref_cnstrnt_notfound: {"Name of Referential Constraint not defined in constraints table.\n", "42000"},
	Isc_foreign_key_notfound: {"Non-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY.\n", "42000"},
	Isc_ref_cnstrnt_update: {"Cannot update constraints (RDB$REF_CONSTRAINTS).\n", "42000"},
	Isc_check_cnstrnt_update: {"Cannot update constraints (RDB$CHECK_CONSTRAINTS).\n", "44000"},
	Isc_check_cnstrnt_del: {"Cannot delete CHECK constraint entry (RDB$CHECK_CONSTRAINTS)\n", "44000"},
	Isc_integ_index_seg_del: {"Cannot delete index segment used by an Integrity Constraint\n", "42000"},
	Isc_integ_index_seg_mod: {"Cannot update index segment used by an Integrity Constraint\n", "42000"},
	Isc_integ_index_del: {"Cannot delete index used by an Integrity Constraint\n", "42000"},
	Isc_integ_index_mod: {"Cannot modify index used by an Integrity Constraint\n", "42000"},
	Isc_check_trig_del: {"Cannot delete trigger used by a CHECK Constraint\n", "44000"},
	Isc_check_trig_update: {"Cannot update trigger used by a CHECK Constraint\n", "44000"},
	Isc_cnstrnt_fld_del: {"Cannot delete column being used in an Integrity Constraint.\n", "23000"},
	Isc_cnstrnt_fld_rename: {"Cannot rename column being used in an Integrity Constraint.\n", "23000"},
	Isc_rel_cnstrnt_update: {"Cannot update constraints (RDB$RELATION_CONSTRAINTS).\n", "42000"},
	Isc_constaint_on_view: {"Cannot define constraints on views\n", "42000"},
	Isc_invld_cnstrnt_type: {"internal Red Database consistency check (invalid RDB$CONSTRAINT_TYPE)\n", "42000"},
	Isc_primary_key_exists: {"Attempt to define a second PRIMARY KEY for the same table\n", "42S11"},
	Isc_systrig_update: {"cannot modify or erase a system trigger\n", "HY000"},
	Isc_not_rel_owner: {"only the owner of a table may reassign ownership\n", "28000"},
	Isc_grant_obj_notfound: {"could not find object for GRANT\n", "01007"},
	Isc_grant_fld_notfound: {"could not find column for GRANT\n", "01007"},
	Isc_grant_nopriv: {"user does not have GRANT privileges for operation\n", "01007"},
	Isc_nonsql_security_rel: {"object has non-SQL security class defined\n", "28000"},
	Isc_nonsql_security_fld: {"column has non-SQL security class defined\n", "28000"},
	Isc_wal_cache_err: {"Write-ahead Log without shared cache configuration not allowed\n", "HY000"},
	Isc_shutfail: {"database shutdown unsuccessful\n", "HY000"},
	Isc_check_constraint: {"Operation violates CHECK constraint @1 on view or table @2\n", "23000"},
	Isc_bad_svc_handle: {"invalid service handle\n", "HY000"},
	Isc_shutwarn: {"database @1 shutdown in @2 seconds\n", "HY000"},
	Isc_wrospbver: {"wrong version of service parameter block\n", "HY000"},
	Isc_bad_spb_form: {"unrecognized service parameter block\n", "42000"},
	Isc_svcnotdef: {"service @1 is not defined\n", "HY000"},
	Isc_no_jrn: {"long-term journaling not enabled\n", "HY000"},
	Isc_transliteration_failed: {"Cannot transliterate character between character sets\n", "22018"},
	Isc_start_cm_for_wal: {"WAL defined; Cache Manager must be started first\n", "HY000"},
	Isc_wal_ovflow_log_required: {"Overflow log specification required for round-robin log\n", "HY000"},
	Isc_text_subtype: {"Implementation of text subtype @1 not located.\n", "42000"},
	Isc_dsql_error: {"Dynamic SQL Error\n", "42000"},
	Isc_dsql_command_err: {"Invalid command\n", "42000"},
	Isc_dsql_constant_err: {"Data type for constant unknown\n", "HY004"},
	Isc_dsql_cursor_err: {"Invalid cursor reference\n", "HY015"},
	Isc_dsql_datatype_err: {"Data type unknown\n", "HY004"},
	Isc_dsql_decl_err: {"Invalid cursor declaration\n", "34000"},
	Isc_dsql_cursor_update_err: {"Cursor @1 is not updatable\n", "24000"},
	Isc_dsql_cursor_open_err: {"Attempt to reopen an open cursor\n", "24000"},
	Isc_dsql_cursor_close_err: {"Attempt to reclose a closed cursor\n", "24000"},
	Isc_dsql_field_err: {"Column unknown\n", "42S22"},
	Isc_dsql_internal_err: {"Internal error\n", "XX000"},
	Isc_dsql_relation_err: {"Table unknown\n", "42S02"},
	Isc_dsql_procedure_err: {"Procedure unknown\n", "42000"},
	Isc_dsql_request_err: {"Request unknown\n", "42000"},
	Isc_dsql_sqlda_err: {"SQLDA error\n", "07002"},
	Isc_dsql_var_count_err: {"Count of read-write columns does not equal count of values\n", "21S01"},
	Isc_dsql_stmt_handle: {"Invalid statement handle\n", "HY000"},
	Isc_dsql_function_err: {"Function unknown\n", "39000"},
	Isc_dsql_blob_err: {"Column is not a BLOB\n", "42000"},
	Isc_collation_not_found: {"COLLATION @1 for CHARACTER SET @2 is not defined\n", "22021"},
	Isc_collation_not_for_charset: {"COLLATION @1 is not valid for specified CHARACTER SET\n", "22021"},
	Isc_dsql_dup_option: {"Option specified more than once\n", "42000"},
	Isc_dsql_tran_err: {"Unknown transaction option\n", "25000"},
	Isc_dsql_invalid_array: {"Invalid array reference\n", "HY000"},
	Isc_dsql_max_arr_dim_exceeded: {"Array declared with too many dimensions\n", "54000"},
	Isc_dsql_arr_range_error: {"Illegal array dimension range\n", "42000"},
	Isc_dsql_trigger_err: {"Trigger unknown\n", "HY000"},
	Isc_dsql_subselect_err: {"Subselect illegal in this context\n", "42000"},
	Isc_dsql_crdb_prepare_err: {"Cannot prepare a CREATE DATABASE statement\n", "42000"},
	Isc_specify_field_err: {"must specify column name for view select expression\n", "42000"},
	Isc_num_field_err: {"number of columns does not match select list\n", "07002"},
	Isc_col_name_err: {"Only simple column names permitted for VIEW WITH CHECK OPTION\n", "42000"},
	Isc_where_err: {"No WHERE clause for VIEW WITH CHECK OPTION\n", "42000"},
	Isc_table_view_err: {"Only one table allowed for VIEW WITH CHECK OPTION\n", "42000"},
	Isc_distinct_err: {"DISTINCT, GROUP or HAVING not permitted for VIEW WITH CHECK OPTION\n", "42000"},
	Isc_key_field_count_err: {"FOREIGN KEY column count does not match PRIMARY KEY\n", "42000"},
	Isc_subquery_err: {"No subqueries permitted for VIEW WITH CHECK OPTION\n", "42000"},
	Isc_expression_eval_err: {"expression evaluation not supported\n", "42000"},
	Isc_node_err: {"gen.c: node not supported\n", "HY000"},
	Isc_command_end_err: {"Unexpected end of command\n", "42000"},
	Isc_index_name: {"INDEX @1\n", "42000"},
	Isc_exception_name: {"EXCEPTION @1\n", "22000"},
	Isc_field_name: {"COLUMN @1\n", "42000"},
	Isc_token_err: {"Token unknown\n", "42000"},
	Isc_union_err: {"union not supported\n", "42000"},
	Isc_dsql_construct_err: {"Unsupported DSQL construct\n", "42000"},
	Isc_field_aggregate_err: {"column used with aggregate\n", "42000"},
	Isc_field_ref_err: {"invalid column reference\n", "42S22"},
	Isc_order_by_err: {"invalid ORDER BY clause\n", "42000"},
	Isc_return_mode_err: {"Return mode by value not allowed for this data type\n", "HY000"},
	Isc_extern_func_err: {"External functions cannot have more than 10 parameters\n", "38000"},
	Isc_alias_conflict_err: {"alias @1 conflicts with an alias in the same statement\n", "42000"},
	Isc_procedure_conflict_error: {"alias @1 conflicts with a procedure in the same statement\n", "42000"},
	Isc_relation_conflict_err: {"alias @1 conflicts with a table in the same statement\n", "42000"},
	Isc_dsql_domain_err: {"Illegal use of keyword VALUE\n", "42000"},
	Isc_idx_seg_err: {"segment count of 0 defined for index @1\n", "42000"},
	Isc_node_name_err: {"A node name is not permitted in a secondary, shadow, cache or log file name\n", "HY000"},
	Isc_table_name: {"TABLE @1\n", "00000"},
	Isc_proc_name: {"PROCEDURE @1\n", "42000"},
	Isc_idx_create_err: {"cannot create index @1\n", "42000"},
	Isc_wal_shadow_err: {"Write-ahead Log with shadowing configuration not allowed\n", "HY000"},
	Isc_dependency: {"there are @1 dependencies\n", "42000"},
	Isc_idx_key_err: {"too many keys defined for index @1\n", "54011"},
	Isc_dsql_file_length_err: {"Preceding file did not specify length, so @1 must include starting page number\n", "42000"},
	Isc_dsql_shadow_number_err: {"Shadow number must be a positive integer\n", "HY000"},
	Isc_dsql_token_unk_err: {"Token unknown - line @1, column @2\n", "42000"},
	Isc_dsql_no_relation_alias: {"there is no alias or table named @1 at this scope level\n", "42S02"},
	Isc_indexname: {"there is no index @1 for table @2\n", "42000"},
	Isc_no_stream_plan: {"table or procedure @1 is not referenced in plan\n", "HY000"},
	Isc_stream_twice: {"table or procedure @1 is referenced more than once in plan; use aliases to distinguish\n", "HY000"},
	Isc_stream_not_found: {"table or procedure @1 is referenced in the plan but not the from list\n", "HY000"},
	Isc_collation_requires_text: {"Invalid use of CHARACTER SET or COLLATE\n", "22021"},
	Isc_dsql_domain_not_found: {"Specified domain or source column @1 does not exist\n", "42000"},
	Isc_index_unused: {"index @1 cannot be used in the specified plan\n", "42000"},
	Isc_dsql_self_join: {"the table @1 is referenced twice; use aliases to differentiate\n", "42000"},
	Isc_stream_bof: {"attempt to fetch before the first record in a record stream\n", "HY109"},
	Isc_stream_crack: {"the current position is on a crack\n", "HY000"},
	Isc_db_or_file_exists: {"database or file exists\n", "HY000"},
	Isc_invalid_operator: {"invalid comparison operator for find operation\n", "42000"},
	Isc_conn_lost: {"Connection lost to pipe server\n", "HY000"},
	Isc_bad_checksum: {"bad checksum\n", "42000"},
	Isc_page_type_err: {"wrong page type\n", "HY000"},
	Isc_ext_readonly_err: {"Cannot insert because the file is readonly or is on a read only medium.\n", "42000"},
	Isc_sing_select_err: {"multiple rows in singleton select\n", "21000"},
	Isc_psw_attach: {"cannot attach to password database\n", "28000"},
	Isc_psw_start_trans: {"cannot start transaction for password database\n", "28000"},
	Isc_invalid_direction: {"invalid direction for find operation\n", "42000"},
	Isc_dsql_var_conflict: {"variable @1 conflicts with parameter in same procedure\n", "HY000"},
	Isc_dsql_no_blob_array: {"Array/BLOB/DATE data types not allowed in arithmetic\n", "HY000"},
	Isc_dsql_base_table: {"@1 is not a valid base table of the specified view\n", "42000"},
	Isc_duplicate_base_table: {"table or procedure @1 is referenced twice in view; use an alias to distinguish\n", "42000"},
	Isc_view_alias: {"view @1 has more than one base table; use aliases to distinguish\n", "42000"},
	Isc_index_root_page_full: {"cannot add index, index root page is full.\n", "54000"},
	Isc_dsql_blob_type_unknown: {"BLOB SUB_TYPE @1 is not defined\n", "42000"},
	Isc_req_max_clones_exceeded: {"Too many concurrent executions of the same request\n", "54001"},
	Isc_dsql_duplicate_spec: {"duplicate specification of @1 - not supported\n", "42000"},
	Isc_unique_key_violation: {"violation of PRIMARY or UNIQUE KEY constraint @1 on table @2\n", "23000"},
	Isc_srvr_version_too_old: {"server version too old to support all CREATE DATABASE options\n", "HY000"},
	Isc_drdb_completed_with_errs: {"drop database completed with errors\n", "HY000"},
	Isc_dsql_procedure_use_err: {"procedure @1 does not return any values\n", "42000"},
	Isc_dsql_count_mismatch: {"count of column list and variable list do not match\n", "07002"},
	Isc_blob_idx_err: {"attempt to index BLOB column in index @1\n", "42000"},
	Isc_array_idx_err: {"attempt to index array column in index @1\n", "42000"},
	Isc_key_field_err: {"too few key columns found for index @1 (incorrect column name?)\n", "42000"},
	Isc_no_delete: {"cannot delete\n", "HY000"},
	Isc_del_last_field: {"last column in a table cannot be deleted\n", "42000"},
	Isc_sort_err: {"sort error\n", "HY000"},
	Isc_sort_mem_err: {"sort error: not enough memory\n", "HY001"},
	Isc_version_err: {"too many versions\n", "54000"},
	Isc_inval_key_posn: {"invalid key position\n", "HY000"},
	Isc_no_segments_err: {"segments not allowed in expression index @1\n", "HY000"},
	Isc_crrp_data_err: {"sort error: corruption in data structure\n", "XX000"},
	Isc_rec_size_err: {"new record size of @1 bytes is too big\n", "54000"},
	Isc_dsql_field_ref: {"Inappropriate self-reference of column\n", "42000"},
	Isc_req_depth_exceeded: {"request depth exceeded. (Recursive definition?)\n", "54001"},
	Isc_no_field_access: {"cannot access column @1 in view @2\n", "42S22"},
	Isc_no_dbkey: {"dbkey not available for multi-table views\n", "HY000"},
	Isc_jrn_format_err: {"journal file wrong format\n", "HY000"},
	Isc_jrn_file_full: {"intermediate journal file full\n", "HY000"},
	Isc_dsql_open_cursor_request: {"The prepare statement identifies a prepare statement with an open cursor\n", "24000"},
	Isc_ib_error: {"Red Database error\n", "HY000"},
	Isc_cache_redef: {"Cache redefined\n", "HY013"},
	Isc_cache_too_small: {"Insufficient memory to allocate page buffer cache\n", "HY013"},
	Isc_log_redef: {"Log redefined\n", "HY000"},
	Isc_log_too_small: {"Log size too small\n", "HY000"},
	Isc_partition_too_small: {"Log partition size too small\n", "HY000"},
	Isc_partition_not_supp: {"Partitions not supported in series of log file specification\n", "HY000"},
	Isc_log_length_spec: {"Total length of a partitioned log must be specified\n", "HY000"},
	Isc_precision_err: {"Precision must be from 1 to 18\n", "HY104"},
	Isc_scale_nogt: {"Scale must be between zero and precision\n", "HY104"},
	Isc_expec_short: {"Short integer expected\n", "22003"},
	Isc_expec_long: {"Long integer expected\n", "22003"},
	Isc_expec_ushort: {"Unsigned short integer expected\n", "22003"},
	Isc_escape_invalid: {"Invalid ESCAPE sequence\n", "22025"},
	Isc_svcnoexe: {"service @1 does not have an associated executable\n", "HY000"},
	Isc_net_lookup_err: {"Failed to locate host machine.\n", "08006"},
	Isc_service_unknown: {"Undefined service @1/@2.\n", "HY000"},
	Isc_host_unknown: {"The specified name was not found in the hosts file or Domain Name Services.\n", "08001"},
	Isc_grant_nopriv_on_base: {"user does not have GRANT privileges on base table/view for operation\n", "01007"},
	Isc_dyn_fld_ambiguous: {"Ambiguous column reference.\n", "42702"},
	Isc_dsql_agg_ref_err: {"Invalid aggregate reference\n", "42000"},
	Isc_complex_view: {"navigational stream @1 references a view with more than one base table\n", "HY000"},
	Isc_unprepared_stmt: {"Attempt to execute an unprepared dynamic SQL statement.\n", "HY007"},
	Isc_expec_positive: {"Positive value expected\n", "42000"},
	Isc_dsql_sqlda_value_err: {"Incorrect values within SQLDA structure\n", "HY000"},
	Isc_invalid_array_id: {"invalid blob id\n", "HY000"},
	Isc_extfile_uns_op: {"Operation not supported for EXTERNAL FILE table @1\n", "42000"},
	Isc_svc_in_use: {"Service is currently busy: @1\n", "HY000"},
	Isc_err_stack_limit: {"stack size insufficient to execute current request\n", "HY013"},
	Isc_invalid_key: {"Invalid key for find operation\n", "42000"},
	Isc_net_init_error: {"Error initializing the network software.\n", "08006"},
	Isc_loadlib_failure: {"Unable to load required library @1.\n", "HY000"},
	Isc_network_error: {"Unable to complete network request to host \"@1\".\n", "08006"},
	Isc_net_connect_err: {"Failed to establish a connection.\n", "08006"},
	Isc_net_connect_listen_err: {"Error while listening for an incoming connection.\n", "08006"},
	Isc_net_event_connect_err: {"Failed to establish a secondary connection for event processing.\n", "08006"},
	Isc_net_event_listen_err: {"Error while listening for an incoming event connection request.\n", "08006"},
	Isc_net_read_err: {"Error reading data from the connection.\n", "08006"},
	Isc_net_write_err: {"Error writing data to the connection.\n", "08006"},
	Isc_integ_index_deactivate: {"Cannot deactivate index used by an integrity constraint\n", "42000"},
	Isc_integ_deactivate_primary: {"Cannot deactivate index used by a PRIMARY/UNIQUE constraint\n", "42000"},
	Isc_cse_not_supported: {"Client/Server Express not supported in this release\n", "HY000"},
	Isc_tra_must_sweep: {"\n", "HY000"},
	Isc_unsupported_network_drive: {"Access to databases on file servers is not supported.\n", "HY000"},
	Isc_io_create_err: {"Error while trying to create file\n", "08001"},
	Isc_io_open_err: {"Error while trying to open file\n", "08001"},
	Isc_io_close_err: {"Error while trying to close file\n", "HY000"},
	Isc_io_read_err: {"Error while trying to read from file\n", "HY000"},
	Isc_io_write_err: {"Error while trying to write to file\n", "HY000"},
	Isc_io_delete_err: {"Error while trying to delete file\n", "HY000"},
	Isc_io_access_err: {"Error while trying to access file\n", "HY000"},
	Isc_udf_exception: {"A fatal exception occurred during the execution of a user defined function.\n", "38000"},
	Isc_lost_db_connection: {"connection lost to database\n", "08006"},
	Isc_no_write_user_priv: {"User cannot write to RDB$USER_PRIVILEGES\n", "28000"},
	Isc_token_too_long: {"token size exceeds limit\n", "42000"},
	Isc_max_att_exceeded: {"Maximum user count exceeded.  Contact your database administrator.\n", "08001"},
	Isc_login_same_as_role_name: {"Your login @1 is same as one of the SQL role name. Ask your database administrator to set up a valid Red Database login.\n", "28000"},
	Isc_reftable_requires_pk: {"\"REFERENCES table\" without \"(column)\" requires PRIMARY KEY on referenced table\n", "42000"},
	Isc_usrname_too_long: {"The username entered is too long.  Maximum length is 31 bytes.\n", "28000"},
	Isc_password_too_long: {"The password specified is too long.  Maximum length is 8 bytes.\n", "28000"},
	Isc_usrname_required: {"A username is required for this operation.\n", "28000"},
	Isc_password_required: {"A password is required for this operation\n", "28000"},
	Isc_bad_protocol: {"The network protocol specified is invalid\n", "42000"},
	Isc_dup_usrname_found: {"A duplicate user name was found in the security database\n", "28000"},
	Isc_usrname_not_found: {"The user name specified was not found in the security database\n", "28000"},
	Isc_error_adding_sec_record: {"An error occurred while attempting to add the user.\n", "28000"},
	Isc_error_modifying_sec_record: {"An error occurred while attempting to modify the user record.\n", "28000"},
	Isc_error_deleting_sec_record: {"An error occurred while attempting to delete the user record.\n", "28000"},
	Isc_error_updating_sec_db: {"An error occurred while updating the security database.\n", "28000"},
	Isc_sort_rec_size_err: {"sort record size of @1 bytes is too big\n", "HY000"},
	Isc_bad_default_value: {"can not define a not null column with NULL as default value\n", "42000"},
	Isc_invalid_clause: {"invalid clause --- '@1'\n", "42000"},
	Isc_too_many_handles: {"too many open handles to database\n", "HY000"},
	Isc_optimizer_blk_exc: {"size of optimizer block exceeded\n", "HY000"},
	Isc_invalid_string_constant: {"a string constant is delimited by double quotes\n", "42000"},
	Isc_transitional_date: {"DATE must be changed to TIMESTAMP\n", "HY000"},
	Isc_read_only_database: {"attempted update on read-only database\n", "42000"},
	Isc_must_be_dialect_2_and_up: {"SQL dialect @1 is not supported in this database\n", "HY000"},
	Isc_blob_filter_exception: {"A fatal exception occurred during the execution of a blob filter.\n", "HY000"},
	Isc_exception_access_violation: {"Access violation.  The code attempted to access a virtual address without privilege to do so.\n", "HY000"},
	Isc_exception_datatype_missalignment: {"Datatype misalignment.  The attempted to read or write a value that was not stored on a memory boundary.\n", "22000"},
	Isc_exception_array_bounds_exceeded: {"Array bounds exceeded.  The code attempted to access an array element that is out of bounds.\n", "22000"},
	Isc_exception_float_denormal_operand: {"Float denormal operand.  One of the floating-point operands is too small to represent a standard float value.\n", "22000"},
	Isc_exception_float_divide_by_zero: {"Floating-point divide by zero.  The code attempted to divide a floating-point value by zero.\n", "22012"},
	Isc_exception_float_inexact_result: {"Floating-point inexact result.  The result of a floating-point operation cannot be represented as a decimal fraction.\n", "22000"},
	Isc_exception_float_invalid_operand: {"Floating-point invalid operand.  An indeterminant error occurred during a floating-point operation.\n", "22000"},
	Isc_exception_float_overflow: {"Floating-point overflow.  The exponent of a floating-point operation is greater than the magnitude allowed.\n", "22003"},
	Isc_exception_float_stack_check: {"Floating-point stack check.  The stack overflowed or underflowed as the result of a floating-point operation.\n", "22000"},
	Isc_exception_float_underflow: {"Floating-point underflow.  The exponent of a floating-point operation is less than the magnitude allowed.\n", "22003"},
	Isc_exception_integer_divide_by_zero: {"Integer divide by zero.  The code attempted to divide an integer value by an integer divisor of zero.\n", "22012"},
	Isc_exception_integer_overflow: {"Integer overflow.  The result of an integer operation caused the most significant bit of the result to carry.\n", "22003"},
	Isc_exception_unknown: {"An exception occurred that does not have a description.  Exception number @1.\n", "HY000"},
	Isc_exception_stack_overflow: {"Stack overflow.  The resource requirements of the runtime stack have exceeded the memory available to it.\n", "HY001"},
	Isc_exception_sigsegv: {"Segmentation Fault. The code attempted to access memory without privileges.\n", "HY000"},
	Isc_exception_sigill: {"Illegal Instruction. The Code attempted to perform an illegal operation.\n", "HY000"},
	Isc_exception_sigbus: {"Bus Error. The Code caused a system bus error.\n", "HY000"},
	Isc_exception_sigfpe: {"Floating Point Error. The Code caused an Arithmetic Exception or a floating point exception.\n", "HY000"},
	Isc_ext_file_delete: {"Cannot delete rows from external files.\n", "42000"},
	Isc_ext_file_modify: {"Cannot update rows in external files.\n", "42000"},
	Isc_adm_task_denied: {"Unable to perform operation\n", "28000"},
	Isc_extract_input_mismatch: {"Specified EXTRACT part does not exist in input datatype\n", "42000"},
	Isc_insufficient_svc_privileges: {"Service @1 requires SYSDBA permissions.  Reattach to the Service Manager using the SYSDBA account.\n", "28000"},
	Isc_file_in_use: {"The file @1 is currently in use by another process.  Try again later.\n", "HY000"},
	Isc_service_att_err: {"Cannot attach to services manager\n", "HY000"},
	Isc_ddl_not_allowed_by_db_sql_dial: {"Metadata update statement is not allowed by the current database SQL dialect @1\n", "42000"},
	Isc_cancelled: {"operation was cancelled\n", "HY008"},
	Isc_unexp_spb_form: {"unexpected item in service parameter block, expected @1\n", "HY000"},
	Isc_sql_dialect_datatype_unsupport: {"Client SQL dialect @1 does not support reference to @2 datatype\n", "HY000"},
	Isc_svcnouser: {"user name and password are required while attaching to the services manager\n", "HY000"},
	Isc_depend_on_uncommitted_rel: {"You created an indirect dependency on uncommitted metadata. You must roll back the current transaction.\n", "42000"},
	Isc_svc_name_missing: {"The service name was not specified.\n", "HY000"},
	Isc_too_many_contexts: {"Too many Contexts of Relation/Procedure/Views. Maximum allowed is 256\n", "54001"},
	Isc_datype_notsup: {"data type not supported for arithmetic\n", "22008"},
	Isc_dialect_reset_warning: {"Database dialect being changed from 3 to 1\n", "01000"},
	Isc_dialect_not_changed: {"Database dialect not changed.\n", "HY000"},
	Isc_database_create_failed: {"Unable to create database @1\n", "HY000"},
	Isc_inv_dialect_specified: {"Database dialect @1 is not a valid dialect.\n", "42000"},
	Isc_valid_db_dialects: {"Valid database dialects are @1.\n", "00000"},
	Isc_sqlwarn: {"SQL warning code = @1\n", "00000"},
	Isc_dtype_renamed: {"DATE data type is now called TIMESTAMP\n", "HY000"},
	Isc_extern_func_dir_error: {"Function @1 is in @2, which is not in a permitted directory for external functions.\n", "38000"},
	Isc_date_range_exceeded: {"value exceeds the range for valid dates\n", "22008"},
	Isc_inv_client_dialect_specified: {"passed client dialect @1 is not a valid dialect.\n", "42000"},
	Isc_valid_client_dialects: {"Valid client dialects are @1.\n", "00000"},
	Isc_optimizer_between_err: {"Unsupported field type specified in BETWEEN predicate.\n", "HY004"},
	Isc_service_not_supported: {"Services functionality will be supported in a later version  of the product\n", "HY000"},
	Isc_generator_name: {"GENERATOR @1\n", "00000"},
	Isc_udf_name: {"Function @1\n", "38000"},
	Isc_bad_limit_param: {"Invalid parameter to FETCH or FIRST. Only integers >= 0 are allowed.\n", "HY000"},
	Isc_bad_skip_param: {"Invalid parameter to OFFSET or SKIP. Only integers >= 0 are allowed.\n", "42000"},
	Isc_io_32bit_exceeded_err: {"File exceeded maximum size of 2GB.  Add another database file or use a 64 bit I/O version of Red Database.\n", "54000"},
	Isc_invalid_savepoint: {"Unable to find savepoint with name @1 in transaction context\n", "3B000"},
	Isc_dsql_column_pos_err: {"Invalid column position used in the @1 clause\n", "42000"},
	Isc_dsql_agg_where_err: {"Cannot use an aggregate or window function in a WHERE clause, use HAVING (for aggregate only) instead\n", "42000"},
	Isc_dsql_agg_group_err: {"Cannot use an aggregate or window function in a GROUP BY clause\n", "42000"},
	Isc_dsql_agg_column_err: {"Invalid expression in the @1 (not contained in either an aggregate function or the GROUP BY clause)\n", "42000"},
	Isc_dsql_agg_having_err: {"Invalid expression in the @1 (neither an aggregate function nor a part of the GROUP BY clause)\n", "42000"},
	Isc_dsql_agg_nested_err: {"Nested aggregate and window functions are not allowed\n", "42000"},
	Isc_exec_sql_invalid_arg: {"Invalid argument in EXECUTE STATEMENT - cannot convert to string\n", "42000"},
	Isc_exec_sql_invalid_req: {"Wrong request type in EXECUTE STATEMENT '@1'\n", "42000"},
	Isc_exec_sql_invalid_var: {"Variable type (position @1) in EXECUTE STATEMENT '@2' INTO does not match returned column type\n", "42000"},
	Isc_exec_sql_max_call_exceeded: {"Too many recursion levels of EXECUTE STATEMENT\n", "42000"},
	Isc_conf_access_denied: {"Use of @1 at location @2 is not allowed by server configuration\n", "28000"},
	Isc_wrong_backup_state: {"Cannot change difference file name while database is in backup mode\n", "HY000"},
	Isc_wal_backup_err: {"Physical backup is not allowed while Write-Ahead Log is in use\n", "HY000"},
	Isc_cursor_not_open: {"Cursor is not open\n", "24000"},
	Isc_bad_shutdown_mode: {"Target shutdown mode is invalid for database \"@1\"\n", "08007"},
	Isc_concat_overflow: {"Concatenation overflow. Resulting string cannot exceed 32765 bytes in length.\n", "HY000"},
	Isc_bad_substring_offset: {"Invalid offset parameter @1 to SUBSTRING. Only positive integers are allowed.\n", "22011"},
	Isc_foreign_key_target_doesnt_exist: {"Foreign key reference target does not exist\n", "42000"},
	Isc_foreign_key_references_present: {"Foreign key references are present for the record\n", "42000"},
	Isc_no_update: {"cannot update\n", "HY000"},
	Isc_cursor_already_open: {"Cursor is already open\n", "24000"},
	Isc_stack_trace: {"@1\n", "00000"},
	Isc_ctx_var_not_found: {"Context variable '@1' is not found in namespace '@2'\n", "HY000"},
	Isc_ctx_namespace_invalid: {"Invalid namespace name '@1' passed to @2\n", "HY000"},
	Isc_ctx_too_big: {"Too many context variables\n", "54000"},
	Isc_ctx_bad_argument: {"Invalid argument passed to @1\n", "HY000"},
	Isc_identifier_too_long: {"BLR syntax error. Identifier @1... is too long\n", "42000"},
	Isc_except2: {"exception @1\n", "HY000"},
	Isc_malformed_string: {"Malformed string\n", "22000"},
	Isc_prc_out_param_mismatch: {"Output parameter mismatch for procedure @1\n", "07002"},
	Isc_command_end_err2: {"Unexpected end of command - line @1, column @2\n", "42000"},
	Isc_partner_idx_incompat_type: {"partner index segment no @1 has incompatible data type\n", "42000"},
	Isc_bad_substring_length: {"Invalid length parameter @1 to SUBSTRING. Negative integers are not allowed.\n", "22011"},
	Isc_charset_not_installed: {"CHARACTER SET @1 is not installed\n", "2C000"},
	Isc_collation_not_installed: {"COLLATION @1 for CHARACTER SET @2 is not installed\n", "22021"},
	Isc_att_shutdown: {"connection shutdown\n", "08003"},
	Isc_blobtoobig: {"Maximum BLOB size exceeded\n", "54000"},
	Isc_must_have_phys_field: {"Can't have relation with only computed fields or constraints\n", "42000"},
	Isc_invalid_time_precision: {"Time precision exceeds allowed range (0-@1)\n", "22007"},
	Isc_blob_convert_error: {"Unsupported conversion to target type BLOB (subtype @1)\n", "HY000"},
	Isc_array_convert_error: {"Unsupported conversion to target type ARRAY\n", "22000"},
	Isc_record_lock_not_supp: {"Stream does not support record locking\n", "40001"},
	Isc_partner_idx_not_found: {"Cannot create foreign key constraint @1. Partner index does not exist or is inactive.\n", "42000"},
	Isc_tra_num_exc: {"Transactions count exceeded. Perform backup and restore to make database operable again\n", "54000"},
	Isc_field_disappeared: {"Column has been unexpectedly deleted\n", "42000"},
	Isc_met_wrong_gtt_scope: {"@1 cannot depend on @2\n", "HY000"},
	Isc_subtype_for_internal_use: {"Blob sub_types bigger than 1 (text) are for internal use only\n", "HY000"},
	Isc_illegal_prc_type: {"Procedure @1 is not selectable (it does not contain a SUSPEND statement)\n", "42000"},
	Isc_invalid_sort_datatype: {"Datatype @1 is not supported for sorting operation\n", "42000"},
	Isc_collation_name: {"COLLATION @1\n", "00000"},
	Isc_domain_name: {"DOMAIN @1\n", "00000"},
	Isc_domnotdef: {"domain @1 is not defined\n", "42000"},
	Isc_array_max_dimensions: {"Array data type can use up to @1 dimensions\n", "54000"},
	Isc_max_db_per_trans_allowed: {"A multi database transaction cannot span more than @1 databases\n", "54000"},
	Isc_bad_debug_format: {"Bad debug info format\n", "00000"},
	Isc_bad_proc_BLR: {"Error while parsing procedure @1's BLR\n", "2F000"},
	Isc_key_too_big: {"index key too big\n", "54000"},
	Isc_concurrent_transaction: {"concurrent transaction number is @1\n", "HY000"},
	Isc_not_valid_for_var: {"validation error for variable @1, value \"@2\"\n", "42000"},
	Isc_not_valid_for: {"validation error for @1, value \"@2\"\n", "42000"},
	Isc_need_difference: {"Difference file name should be set explicitly for database on raw device\n", "HY000"},
	Isc_long_login: {"Login name too long (@1 characters, maximum allowed @2)\n", "HY000"},
	Isc_fldnotdef2: {"column @1 is not defined in procedure @2\n", "42S22"},
	Isc_invalid_similar_pattern: {"Invalid SIMILAR TO pattern\n", "42000"},
	Isc_bad_teb_form: {"Invalid TEB format\n", "HY000"},
	Isc_tpb_multiple_txn_isolation: {"Found more than one transaction isolation in TPB\n", "HY000"},
	Isc_tpb_reserv_before_table: {"Table reservation lock type @1 requires table name before in TPB\n", "HY000"},
	Isc_tpb_multiple_spec: {"Found more than one @1 specification in TPB\n", "HY000"},
	Isc_tpb_option_without_rc: {"Option @1 requires READ COMMITTED isolation in TPB\n", "HY000"},
	Isc_tpb_conflicting_options: {"Option @1 is not valid if @2 was used previously in TPB\n", "HY000"},
	Isc_tpb_reserv_missing_tlen: {"Table name length missing after table reservation @1 in TPB\n", "HY000"},
	Isc_tpb_reserv_long_tlen: {"Table name length @1 is too long after table reservation @2 in TPB\n", "HY000"},
	Isc_tpb_reserv_missing_tname: {"Table name length @1 without table name after table reservation @2 in TPB\n", "HY000"},
	Isc_tpb_reserv_corrup_tlen: {"Table name length @1 goes beyond the remaining TPB size after table reservation @2\n", "HY000"},
	Isc_tpb_reserv_null_tlen: {"Table name length is zero after table reservation @1 in TPB\n", "HY000"},
	Isc_tpb_reserv_relnotfound: {"Table or view @1 not defined in system tables after table reservation @2 in TPB\n", "HY000"},
	Isc_tpb_reserv_baserelnotfound: {"Base table or view @1 for view @2 not defined in system tables after table reservation @3 in TPB\n", "HY000"},
	Isc_tpb_missing_len: {"Option length missing after option @1 in TPB\n", "HY000"},
	Isc_tpb_missing_value: {"Option length @1 without value after option @2 in TPB\n", "HY000"},
	Isc_tpb_corrupt_len: {"Option length @1 goes beyond the remaining TPB size after option @2\n", "HY000"},
	Isc_tpb_null_len: {"Option length is zero after table reservation @1 in TPB\n", "HY000"},
	Isc_tpb_overflow_len: {"Option length @1 exceeds the range for option @2 in TPB\n", "HY000"},
	Isc_tpb_invalid_value: {"Option value @1 is invalid for the option @2 in TPB\n", "HY000"},
	Isc_tpb_reserv_stronger_wng: {"Preserving previous table reservation @1 for table @2, stronger than new @3 in TPB\n", "HY000"},
	Isc_tpb_reserv_stronger: {"Table reservation @1 for table @2 already specified and is stronger than new @3 in TPB\n", "HY000"},
	Isc_tpb_reserv_max_recursion: {"Table reservation reached maximum recursion of @1 when expanding views in TPB\n", "HY000"},
	Isc_tpb_reserv_virtualtbl: {"Table reservation in TPB cannot be applied to @1 because it's a virtual table\n", "HY000"},
	Isc_tpb_reserv_systbl: {"Table reservation in TPB cannot be applied to @1 because it's a system table\n", "HY000"},
	Isc_tpb_reserv_temptbl: {"Table reservation @1 or @2 in TPB cannot be applied to @3 because it's a temporary table\n", "HY000"},
	Isc_tpb_readtxn_after_writelock: {"Cannot set the transaction in read only mode after a table reservation isc_tpb_lock_write in TPB\n", "HY000"},
	Isc_tpb_writelock_after_readtxn: {"Cannot take a table reservation isc_tpb_lock_write in TPB because the transaction is in read only mode\n", "HY000"},
	Isc_time_range_exceeded: {"value exceeds the range for a valid time\n", "22008"},
	Isc_datetime_range_exceeded: {"value exceeds the range for valid timestamps\n", "22008"},
	Isc_string_truncation: {"string right truncation\n", "22001"},
	Isc_blob_truncation: {"blob truncation when converting to a string: length limit exceeded\n", "22001"},
	Isc_numeric_out_of_range: {"numeric value is out of range\n", "22003"},
	Isc_shutdown_timeout: {"Red Database shutdown is still in progress after the specified timeout\n", "HYT00"},
	Isc_att_handle_busy: {"Attachment handle is busy\n", "HY010"},
	Isc_bad_udf_freeit: {"Bad written UDF detected: pointer returned in FREE_IT function was not allocated by ib_util_malloc\n", "39000"},
	Isc_eds_provider_not_found: {"External Data Source provider '@1' not found\n", "42000"},
	Isc_eds_connection: {"Execute statement error at @1 :\n@2Data source : @3\n", "42000"},
	Isc_eds_preprocess: {"Execute statement preprocess SQL error\n", "42000"},
	Isc_eds_stmt_expected: {"Statement expected\n", "42000"},
	Isc_eds_prm_name_expected: {"Parameter name expected\n", "42000"},
	Isc_eds_unclosed_comment: {"Unclosed comment found near '@1'\n", "42000"},
	Isc_eds_statement: {"Execute statement error at @1 :\n@2Statement : @3\nData source : @4\n", "42000"},
	Isc_eds_input_prm_mismatch: {"Input parameters mismatch\n", "42000"},
	Isc_eds_output_prm_mismatch: {"Output parameters mismatch\n", "42000"},
	Isc_eds_input_prm_not_set: {"Input parameter '@1' have no value set\n", "42000"},
	Isc_too_big_blr: {"BLR stream length @1 exceeds implementation limit @2\n", "HY000"},
	Isc_montabexh: {"Monitoring table space exhausted\n", "HY001"},
	Isc_modnotfound: {"module name or entrypoint could not be found\n", "39000"},
	Isc_nothing_to_cancel: {"nothing to cancel\n", "HY018"},
	Isc_ibutil_not_loaded: {"ib_util library has not been loaded to deallocate memory returned by FREE_IT function\n", "39000"},
	Isc_circular_computed: {"Cannot have circular dependencies with computed fields\n", "42000"},
	Isc_psw_db_error: {"Security database error\n", "28000"},
	Isc_invalid_type_datetime_op: {"Invalid data type in DATE/TIME/TIMESTAMP addition or subtraction in add_datettime()\n", "42000"},
	Isc_onlycan_add_timetodate: {"Only a TIME value can be added to a DATE value\n", "42000"},
	Isc_onlycan_add_datetotime: {"Only a DATE value can be added to a TIME value\n", "42000"},
	Isc_onlycansub_tstampfromtstamp: {"TIMESTAMP values can be subtracted only from another TIMESTAMP value\n", "42000"},
	Isc_onlyoneop_mustbe_tstamp: {"Only one operand can be of type TIMESTAMP\n", "42000"},
	Isc_invalid_extractpart_time: {"Only HOUR, MINUTE, SECOND and MILLISECOND can be extracted from TIME values\n", "42000"},
	Isc_invalid_extractpart_date: {"HOUR, MINUTE, SECOND and MILLISECOND cannot be extracted from DATE values\n", "42000"},
	Isc_invalidarg_extract: {"Invalid argument for EXTRACT() not being of DATE/TIME/TIMESTAMP type\n", "42000"},
	Isc_sysf_argmustbe_exact: {"Arguments for @1 must be integral types or NUMERIC/DECIMAL without scale\n", "42000"},
	Isc_sysf_argmustbe_exact_or_fp: {"First argument for @1 must be integral type or floating point type\n", "42000"},
	Isc_sysf_argviolates_uuidtype: {"Human readable UUID argument for @1 must be of string type\n", "42000"},
	Isc_sysf_argviolates_uuidlen: {"Human readable UUID argument for @2 must be of exact length @1\n", "42000"},
	Isc_sysf_argviolates_uuidfmt: {"Human readable UUID argument for @3 must have \"-\" at position @2 instead of \"@1\"\n", "42000"},
	Isc_sysf_argviolates_guidigits: {"Human readable UUID argument for @3 must have hex digit at position @2 instead of \"@1\"\n", "42000"},
	Isc_sysf_invalid_addpart_time: {"Only HOUR, MINUTE, SECOND and MILLISECOND can be added to TIME values in @1\n", "42000"},
	Isc_sysf_invalid_add_datetime: {"Invalid data type in addition of part to DATE/TIME/TIMESTAMP in @1\n", "42000"},
	Isc_sysf_invalid_addpart_dtime: {"Invalid part @1 to be added to a DATE/TIME/TIMESTAMP value in @2\n", "42000"},
	Isc_sysf_invalid_add_dtime_rc: {"Expected DATE/TIME/TIMESTAMP type in evlDateAdd() result\n", "42000"},
	Isc_sysf_invalid_diff_dtime: {"Expected DATE/TIME/TIMESTAMP type as first and second argument to @1\n", "42000"},
	Isc_sysf_invalid_timediff: {"The result of TIME-<value> in @1 cannot be expressed in YEAR, MONTH, DAY or WEEK\n", "42000"},
	Isc_sysf_invalid_tstamptimediff: {"The result of TIME-TIMESTAMP or TIMESTAMP-TIME in @1 cannot be expressed in HOUR, MINUTE, SECOND or MILLISECOND\n", "42000"},
	Isc_sysf_invalid_datetimediff: {"The result of DATE-TIME or TIME-DATE in @1 cannot be expressed in HOUR, MINUTE, SECOND and MILLISECOND\n", "42000"},
	Isc_sysf_invalid_diffpart: {"Invalid part @1 to express the difference between two DATE/TIME/TIMESTAMP values in @2\n", "42000"},
	Isc_sysf_argmustbe_positive: {"Argument for @1 must be positive\n", "42000"},
	Isc_sysf_basemustbe_positive: {"Base for @1 must be positive\n", "42000"},
	Isc_sysf_argnmustbe_nonneg: {"Argument #@1 for @2 must be zero or positive\n", "42000"},
	Isc_sysf_argnmustbe_positive: {"Argument #@1 for @2 must be positive\n", "42000"},
	Isc_sysf_invalid_zeropowneg: {"Base for @1 cannot be zero if exponent is negative\n", "42000"},
	Isc_sysf_invalid_negpowfp: {"Base for @1 cannot be negative if exponent is not an integral value\n", "42000"},
	Isc_sysf_invalid_scale: {"The numeric scale must be between -128 and 127 in @1\n", "42000"},
	Isc_sysf_argmustbe_nonneg: {"Argument for @1 must be zero or positive\n", "42000"},
	Isc_sysf_binuuid_mustbe_str: {"Binary UUID argument for @1 must be of string type\n", "42000"},
	Isc_sysf_binuuid_wrongsize: {"Binary UUID argument for @2 must use @1 bytes\n", "42000"},
	Isc_missing_required_spb: {"Missing required item @1 in service parameter block\n", "HY000"},
	Isc_net_server_shutdown: {"@1 server is shutdown\n", "08006"},
	Isc_bad_conn_str: {"Invalid connection string\n", "08001"},
	Isc_bad_epb_form: {"Unrecognized events block\n", "HY000"},
	Isc_no_threads: {"Could not start first worker thread - shutdown server\n", "08004"},
	Isc_net_event_connect_timeout: {"Timeout occurred while waiting for a secondary connection for event processing\n", "08006"},
	Isc_sysf_argmustbe_nonzero: {"Argument for @1 must be different than zero\n", "42000"},
	Isc_sysf_argmustbe_range_inc1_1: {"Argument for @1 must be in the range [-1, 1]\n", "42000"},
	Isc_sysf_argmustbe_gteq_one: {"Argument for @1 must be greater or equal than one\n", "42000"},
	Isc_sysf_argmustbe_range_exc1_1: {"Argument for @1 must be in the range ]-1, 1[\n", "42000"},
	Isc_internal_rejected_params: {"Incorrect parameters provided to internal function @1\n", "HY000"},
	Isc_sysf_fp_overflow: {"Floating point overflow in built-in function @1\n", "42000"},
	Isc_udf_fp_overflow: {"Floating point overflow in result from UDF @1\n", "39000"},
	Isc_udf_fp_nan: {"Invalid floating point value returned by UDF @1\n", "39000"},
	Isc_instance_conflict: {"Shared memory area is probably already created by another engine instance in another Windows session\n", "42000"},
	Isc_out_of_temp_space: {"No free space found in temporary directories\n", "HY000"},
	Isc_eds_expl_tran_ctrl: {"Explicit transaction control is not allowed\n", "42000"},
	Isc_no_trusted_spb: {"Use of TRUSTED switches in spb_command_line is prohibited\n", "28000"},
	Isc_package_name: {"PACKAGE @1\n", "42000"},
	Isc_cannot_make_not_null: {"Cannot make field @1 of table @2 NOT NULL because there are NULLs present\n", "22006"},
	Isc_feature_removed: {"Feature @1 is not supported anymore\n", "0A000"},
	Isc_view_name: {"VIEW @1\n", "00000"},
	Isc_lock_dir_access: {"Can not access lock files directory @1\n", "08006"},
	Isc_invalid_fetch_option: {"Fetch option @1 is invalid for a non-scrollable cursor\n", "HY106"},
	Isc_bad_fun_BLR: {"Error while parsing function @1's BLR\n", "2F000"},
	Isc_func_pack_not_implemented: {"Cannot execute function @1 of the unimplemented package @2\n", "2F000"},
	Isc_proc_pack_not_implemented: {"Cannot execute procedure @1 of the unimplemented package @2\n", "2F000"},
	Isc_eem_func_not_returned: {"External function @1 not returned by the external engine plugin @2\n", "39000"},
	Isc_eem_proc_not_returned: {"External procedure @1 not returned by the external engine plugin @2\n", "39000"},
	Isc_eem_trig_not_returned: {"External trigger @1 not returned by the external engine plugin @2\n", "39000"},
	Isc_eem_bad_plugin_ver: {"Incompatible plugin version @1 for external engine @2\n", "39000"},
	Isc_eem_engine_notfound: {"External engine @1 not found\n", "39000"},
	Isc_attachment_in_use: {"Attachment is in use\n", "08002"},
	Isc_transaction_in_use: {"Transaction is in use\n", "25S02"},
	Isc_pman_cannot_load_plugin: {"Error loading plugin @1\n", "39000"},
	Isc_pman_module_notfound: {"Loadable module @1 not found\n", "00000"},
	Isc_pman_entrypoint_notfound: {"Standard plugin entrypoint does not exist in module @1\n", "00000"},
	Isc_pman_module_bad: {"Module @1 exists but can not be loaded\n", "00000"},
	Isc_pman_plugin_notfound: {"Module @1 does not contain plugin @2 type @3\n", "00000"},
	Isc_sysf_invalid_trig_namespace: {"Invalid usage of context namespace DDL_TRIGGER\n", "42000"},
	Isc_unexpected_null: {"Value is NULL but isNull parameter was not informed\n", "22002"},
	Isc_type_notcompat_blob: {"Type @1 is incompatible with BLOB\n", "2200G"},
	Isc_invalid_date_val: {"Invalid date\n", "22003"},
	Isc_invalid_time_val: {"Invalid time\n", "22003"},
	Isc_invalid_timestamp_val: {"Invalid timestamp\n", "22003"},
	Isc_invalid_index_val: {"Invalid index @1 in function @2\n", "22003"},
	Isc_formatted_exception: {"@1\n", "00000"},
	Isc_async_active: {"Asynchronous call is already running for this attachment\n", "HY018"},
	Isc_private_function: {"Function @1 is private to package @2\n", "42000"},
	Isc_private_procedure: {"Procedure @1 is private to package @2\n", "42000"},
	Isc_request_outdated: {"Request can't access new records in relation @1 and should be recompiled\n", "24000"},
	Isc_bad_events_handle: {"invalid events id (handle)\n", "HY000"},
	Isc_cannot_copy_stmt: {"Cannot copy statement @1\n", "XX000"},
	Isc_invalid_boolean_usage: {"Invalid usage of boolean expression\n", "22000"},
	Isc_sysf_argscant_both_be_zero: {"Arguments for @1 cannot both be zero\n", "42000"},
	Isc_spb_no_id: {"missing service ID in spb\n", "HY000"},
	Isc_ee_blr_mismatch_null: {"External BLR message mismatch: invalid null descriptor at field @1\n", "42000"},
	Isc_ee_blr_mismatch_length: {"External BLR message mismatch: length = @1, expected @2\n", "42000"},
	Isc_ss_out_of_bounds: {"Subscript @1 out of bounds [@2, @3]\n", "42000"},
	Isc_missing_data_structures: {"Install incomplete. To complete security database initialization please CREATE USER. For details read doc/README.security_database.txt.\n", "28000"},
	Isc_protect_sys_tab: {"@1 operation is not allowed for system table @2\n", "42000"},
	Isc_libtommath_generic: {"Libtommath error code @1 in function @2\n", "2F000"},
	Isc_wroblrver2: {"unsupported BLR version (expected between @1 and @2, encountered @3)\n", "HY000"},
	Isc_trunc_limits: {"expected length @1, actual @2\n", "28000"},
	Isc_info_access: {"Wrong info requested in isc_svc_query() for anonymous service\n", "28000"},
	Isc_svc_no_stdin: {"No isc_info_svc_stdin in user request, but service thread requested stdin data\n", "HY024"},
	Isc_svc_start_failed: {"Start request for anonymous service is impossible\n", "28000"},
	Isc_svc_no_switches: {"All services except for getting server log require switches\n", "HY024"},
	Isc_svc_bad_size: {"Size of stdin data is more than was requested from client\n", "HY109"},
	Isc_no_crypt_plugin: {"Crypt plugin @1 failed to load\n", "HY024"},
	Isc_cp_name_too_long: {"Length of crypt plugin name should not exceed @1 bytes\n", "22001"},
	Isc_cp_process_active: {"Crypt failed - already crypting database\n", "42818"},
	Isc_cp_already_crypted: {"Crypt failed - database is already in requested state\n", "42818"},
	Isc_decrypt_error: {"Missing crypt plugin, but page appears encrypted\n", "XX000"},
	Isc_no_providers: {"No providers loaded\n", "39000"},
	Isc_null_spb: {"NULL data with non-zero SPB length\n", "42818"},
	Isc_max_args_exceeded: {"Maximum (@1) number of arguments exceeded for function @2\n", "42000"},
	Isc_ee_blr_mismatch_names_count: {"External BLR message mismatch: names count = @1, blr count = @2\n", "42000"},
	Isc_ee_blr_mismatch_name_not_found: {"External BLR message mismatch: name @1 not found\n", "42000"},
	Isc_bad_result_set: {"Invalid resultset interface\n", "26000"},
	Isc_wrong_message_length: {"Message length passed from user application does not match set of columns\n", "07000"},
	Isc_no_output_format: {"Resultset is missing output format information\n", "07000"},
	Isc_item_finish: {"Message metadata not ready - item @1 is not finished\n", "HY021"},
	Isc_miss_config: {"Missing configuration file: @1\n", "XX000"},
	Isc_conf_line: {"@1: illegal line <@2>\n", "XX000"},
	Isc_conf_include: {"Invalid include operator in @1 for <@2>\n", "XX000"},
	Isc_include_depth: {"Include depth too big\n", "XX000"},
	Isc_include_miss: {"File to include not found\n", "XX000"},
	Isc_protect_ownership: {"Only the owner can change the ownership\n", "28000"},
	Isc_badvarnum: {"undefined variable number\n", "HY000"},
	Isc_sec_context: {"Missing security context for @1\n", "28000"},
	Isc_multi_segment: {"Missing segment @1 in multisegment connect block parameter\n", "28000"},
	Isc_login_changed: {"Different logins in connect and attach packets - client library error\n", "28000"},
	Isc_auth_handshake_limit: {"Exceeded exchange limit during authentication handshake\n", "28000"},
	Isc_wirecrypt_incompatible: {"Incompatible wire encryption levels requested on client and server\n", "28000"},
	Isc_miss_wirecrypt: {"Client attempted to attach unencrypted but wire encryption is required\n", "28000"},
	Isc_wirecrypt_key: {"Client attempted to start wire encryption using unknown key @1\n", "28000"},
	Isc_wirecrypt_plugin: {"Client attempted to start wire encryption using unsupported plugin @1\n", "28000"},
	Isc_secdb_name: {"Error getting security database name from configuration file\n", "28000"},
	Isc_auth_data: {"Client authentication plugin is missing required data from server\n", "28000"},
	Isc_auth_datalength: {"Client authentication plugin expected @2 bytes of @3 from server, got @1\n", "28000"},
	Isc_info_unprepared_stmt: {"Attempt to get information about an unprepared dynamic SQL statement.\n", "HY007"},
	Isc_idx_key_value: {"Problematic key value is @1\n", "HY007"},
	Isc_forupdate_virtualtbl: {"Cannot select virtual table @1 for update WITH LOCK\n", "HY000"},
	Isc_forupdate_systbl: {"Cannot select system table @1 for update WITH LOCK\n", "HY000"},
	Isc_forupdate_temptbl: {"Cannot select temporary table @1 for update WITH LOCK\n", "HY000"},
	Isc_cant_modify_sysobj: {"System @1 @2 cannot be modified\n", "42000"},
	Isc_server_misconfigured: {"Server misconfigured - contact administrator please\n", "08004"},
	Isc_alter_role: {"Deprecated backward compatibility ALTER ROLE ... SET/DROP AUTO ADMIN mapping may be used only for RDB$ADMIN role\n", "0A000"},
	Isc_map_already_exists: {"Mapping @1 already exists\n", "42S01"},
	Isc_map_not_exists: {"Mapping @1 does not exist\n", "42S02"},
	Isc_map_load: {"@1 failed when loading mapping cache\n", "08004"},
	Isc_map_aster: {"Invalid name <*> in authentication block\n", "08004"},
	Isc_map_multi: {"Multiple maps found for @1\n", "08004"},
	Isc_map_undefined: {"Undefined mapping result - more than one different results found\n", "08004"},
	Isc_baddpb_damaged_mode: {"Incompatible mode of attachment to damaged database\n", "HY000"},
	Isc_baddpb_buffers_range: {"Attempt to set in database number of buffers which is out of acceptable range [@1:@2]\n", "HY000"},
	Isc_baddpb_temp_buffers: {"Attempt to temporarily set number of buffers less than @1\n", "HY000"},
	Isc_map_nodb: {"Global mapping is not available when database @1 is not present\n", "0A000"},
	Isc_map_notable: {"Global mapping is not available when table RDB$MAP is not present in database @1\n", "0A000"},
	Isc_miss_trusted_role: {"Your attachment has no trusted role\n", "0P000"},
	Isc_set_invalid_role: {"Role @1 is invalid or unavailable\n", "0P000"},
	Isc_cursor_not_positioned: {"Cursor @1 is not positioned in a valid record\n", "HY109"},
	Isc_dup_attribute: {"Duplicated user attribute @1\n", "42702"},
	Isc_dyn_no_priv: {"There is no privilege for this operation\n", "42000"},
	Isc_dsql_cant_grant_option: {"Using GRANT OPTION on @1 not allowed\n", "42000"},
	Isc_read_conflict: {"read conflicts with concurrent update\n", "40001"},
	Isc_crdb_load: {"@1 failed when working with CREATE DATABASE grants\n", "08004"},
	Isc_crdb_nodb: {"CREATE DATABASE grants check is not possible when database @1 is not present\n", "0A000"},
	Isc_crdb_notable: {"CREATE DATABASE grants check is not possible when table RDB$DB_CREATORS is not present in database @1\n", "0A000"},
	Isc_interface_version_too_old: {"Interface @3 version too old: expected @1, found @2\n", "HY000"},
	Isc_fun_param_mismatch: {"Parameter mismatch for function @1\n", "07001"},
	Isc_savepoint_backout_err: {"Error during savepoint backout - transaction invalidated\n", "HY000"},
	Isc_domain_primary_key_notnull: {"Domain used in the PRIMARY KEY constraint of table @1 must be NOT NULL\n", "42000"},
	Isc_invalid_attachment_charset: {"CHARACTER SET @1 cannot be used as a attachment character set\n", "2C000"},
	Isc_map_down: {"Some database(s) were shutdown when trying to read mapping data\n", "08004"},
	Isc_login_error: {"Error occurred during login, please check server rdbserver.log for details\n", "08006"},
	Isc_already_opened: {"Database already opened with engine instance, incompatible with current\n", "08006"},
	Isc_bad_crypt_key: {"Invalid crypt key @1\n", "08006"},
	Isc_encrypt_error: {"Page requires encryption but crypt plugin is missing\n", "XX000"},
	Isc_max_idx_depth: {"Maximum index depth (@1 levels) is reached\n", "54000"},
	Isc_wrong_prvlg: {"System privilege @1 does not exist\n", "0A000"},
	Isc_miss_prvlg: {"System privilege @1 is missing\n", "28000"},
	Isc_crypt_checksum: {"Invalid or missing checksum of encrypted database\n", "XX000"},
	Isc_not_dba: {"You must have SYSDBA rights at this server\n", "28000"},
	Isc_no_cursor: {"Cannot open cursor for non-SELECT statement\n", "07005"},
	Isc_dsql_window_incompat_frames: {"If <window frame bound 1> specifies @1, then <window frame bound 2> shall not specify @2\n", "42000"},
	Isc_dsql_window_range_multi_key: {"RANGE based window with <expr> {PRECEDING | FOLLOWING} cannot have ORDER BY with more than one value\n", "42000"},
	Isc_dsql_window_range_inv_key_type: {"RANGE based window with <offset> PRECEDING/FOLLOWING must have a single ORDER BY key of numerical, date, time or timestamp types\n", "42000"},
	Isc_dsql_window_frame_value_inv_type: {"Window RANGE/ROWS PRECEDING/FOLLOWING value must be of a numerical type\n", "42000"},
	Isc_window_frame_value_invalid: {"Invalid PRECEDING or FOLLOWING offset in window function: cannot be negative\n", "42000"},
	Isc_dsql_window_not_found: {"Window @1 not found\n", "42000"},
	Isc_dsql_window_cant_overr_part: {"Cannot use PARTITION BY clause while overriding the window @1\n", "42000"},
	Isc_dsql_window_cant_overr_order: {"Cannot use ORDER BY clause while overriding the window @1 which already has an ORDER BY clause\n", "42000"},
	Isc_dsql_window_cant_overr_frame: {"Cannot override the window @1 because it has a frame clause. Tip: it can be used without parenthesis in OVER\n", "42000"},
	Isc_dsql_window_duplicate: {"Duplicate window definition for @1\n", "42000"},
	Isc_sql_too_long: {"SQL statement is too long. Maximum size is @1 bytes.\n", "54001"},
	Isc_cfg_stmt_timeout: {"Config level timeout expired.\n", "HY008"},
	Isc_att_stmt_timeout: {"Attachment level timeout expired.\n", "HY008"},
	Isc_req_stmt_timeout: {"Statement level timeout expired.\n", "HY008"},
	Isc_att_shut_killed: {"Killed by database administrator.\n", "08003"},
	Isc_att_shut_idle: {"Idle timeout expired.\n", "08003"},
	Isc_att_shut_db_down: {"Database is shutdown.\n", "08003"},
	Isc_att_shut_engine: {"Engine is shutdown.\n", "08003"},
	Isc_overriding_without_identity: {"OVERRIDING clause can be used only when an identity column is present in the INSERT's field list for table/view @1\n", "42000"},
	Isc_overriding_system_invalid: {"OVERRIDING SYSTEM VALUE can be used only for identity column defined as 'GENERATED ALWAYS' in INSERT for table/view @1\n", "42000"},
	Isc_overriding_user_invalid: {"OVERRIDING USER VALUE can be used only for identity column defined as 'GENERATED BY DEFAULT' in INSERT for table/view @1\n", "42000"},
	Isc_overriding_missing: {"OVERRIDING clause should be used when an identity column defined as 'GENERATED ALWAYS' is present in the INSERT's field list for table table/view @1\n", "42000"},
	Isc_decprecision_err: {"DecFloat precision must be 16 or 34\n", "HY104"},
	Isc_decfloat_divide_by_zero: {"Decimal float divide by zero.  The code attempted to divide a DECFLOAT value by zero.\n", "22012"},
	Isc_decfloat_inexact_result: {"Decimal float inexact result.  The result of an operation cannot be represented as a decimal fraction.\n", "22000"},
	Isc_decfloat_invalid_operation: {"Decimal float invalid operation.  An indeterminant error occurred during an operation.\n", "22000"},
	Isc_decfloat_overflow: {"Decimal float overflow.  The exponent of a result is greater than the magnitude allowed.\n", "22003"},
	Isc_decfloat_underflow: {"Decimal float underflow.  The exponent of a result is less than the magnitude allowed.\n", "22003"},
	Isc_subfunc_notdef: {"Sub-function @1 has not been defined\n", "42000"},
	Isc_subproc_notdef: {"Sub-procedure @1 has not been defined\n", "42000"},
	Isc_subfunc_signat: {"Sub-function @1 has a signature mismatch with its forward declaration\n", "42000"},
	Isc_subproc_signat: {"Sub-procedure @1 has a signature mismatch with its forward declaration\n", "42000"},
	Isc_subfunc_defvaldecl: {"Default values for parameters are not allowed in definition of the previously declared sub-function @1\n", "42000"},
	Isc_subproc_defvaldecl: {"Default values for parameters are not allowed in definition of the previously declared sub-procedure @1\n", "42000"},
	Isc_subfunc_not_impl: {"Sub-function @1 was declared but not implemented\n", "42000"},
	Isc_subproc_not_impl: {"Sub-procedure @1 was declared but not implemented\n", "42000"},
	Isc_sysf_invalid_hash_algorithm: {"Invalid HASH algorithm @1\n", "42000"},
	Isc_expression_eval_index: {"Expression evaluation error for index @1 on table @2\n", "42000"},
	Isc_invalid_decfloat_trap: {"Invalid decfloat trap state @1\n", "42000"},
	Isc_invalid_decfloat_round: {"Invalid decfloat rounding mode @1\n", "42000"},
	Isc_sysf_invalid_first_last_part: {"Invalid part @1 to calculate the @1 of a DATE/TIMESTAMP\n", "42000"},
	Isc_sysf_invalid_date_timestamp: {"Expected DATE/TIMESTAMP value in @1\n", "42000"},
	Isc_precision_err2: {"Precision must be from @1 to @2\n", "HY104"},
	Isc_bad_batch_handle: {"invalid batch handle\n", "HY000"},
	Isc_intl_char: {"Bad international character in tag @1\n", "22021"},
	Isc_null_block: {"Null data in parameters block with non-zero length\n", "22000"},
	Isc_mixed_info: {"Items working with running service and getting generic server information should not be mixed in single info block\n", "22000"},
	Isc_unknown_info: {"Unknown information item, code @1\n", "22000"},
	Isc_bpb_version: {"Wrong version of blob parameters block @1, should be @2\n", "22000"},
	Isc_user_manager: {"User management plugin is missing or failed to load\n", "28000"},
	Isc_icu_entrypoint: {"Missing entrypoint @1 in ICU library\n", "XX000"},
	Isc_icu_library: {"Could not find acceptable ICU library\n", "XX000"},
	Isc_metadata_name: {"Name @1 not found in system MetadataBuilder\n", "02000"},
	Isc_tokens_parse: {"Parse to tokens error\n", "42000"},
	Isc_iconv_open: {"Error opening international conversion descriptor from @1 to @2\n", "08006"},
	Isc_batch_compl_range: {"Message @1 is out of range, only @2 messages in batch\n", "07002"},
	Isc_batch_compl_detail: {"Detailed error info for message @1 is missing in batch\n", "07002"},
	Isc_deflate_init: {"Compression stream init error @1\n", "08006"},
	Isc_inflate_init: {"Decompression stream init error @1\n", "08006"},
	Isc_big_segment: {"Segment size (@1) should not exceed 65535 (64K - 1) when using segmented blob\n", "22026"},
	Isc_batch_policy: {"Invalid blob policy in the batch for @1() call\n", "22000"},
	Isc_batch_defbpb: {"Can't change default BPB after adding any data to batch\n", "22000"},
	Isc_batch_align: {"Unexpected info buffer structure querying for server batch parameters\n", "22000"},
	Isc_multi_segment_dup: {"Duplicated segment @1 in multisegment connect block parameter\n", "22000"},
	Isc_non_plugin_protocol: {"Plugin not supported by network protocol\n", "08004"},
	Isc_message_format: {"Error parsing message format\n", "22000"},
	Isc_batch_param_version: {"Wrong version of batch parameters block @1, should be @2\n", "22000"},
	Isc_batch_msg_long: {"Message size (@1) in batch exceeds internal buffer size (@2)\n", "54000"},
	Isc_batch_open: {"Batch already opened for this statement\n", "HY010"},
	Isc_batch_type: {"Invalid type of statement used in batch\n", "07003"},
	Isc_batch_param: {"Statement used in batch must have parameters\n", "07001"},
	Isc_batch_blobs: {"There are no blobs in associated with batch statement\n", "07001"},
	Isc_batch_blob_append: {"appendBlobData() is used to append data to last blob but no such blob was added to the batch\n", "22000"},
	Isc_batch_stream_align: {"Portions of data, passed as blob stream, should have size multiple to the alignment required for blobs\n", "22000"},
	Isc_batch_rpt_blob: {"Repeated blob id @1 in registerBlob()\n", "22000"},
	Isc_batch_blob_buf: {"Blob buffer format error\n", "22000"},
	Isc_batch_small_data: {"Unusable (too small) data remained in @1 buffer\n", "22000"},
	Isc_batch_cont_bpb: {"Blob continuation should not contain BPB\n", "22000"},
	Isc_batch_big_bpb: {"Size of BPB (@1) greater than remaining data (@2)\n", "22000"},
	Isc_batch_big_segment: {"Size of segment (@1) greater than current BLOB data (@2)\n", "22000"},
	Isc_batch_big_seg2: {"Size of segment (@1) greater than available data (@2)\n", "22000"},
	Isc_batch_blob_id: {"Unknown blob ID @1 in the batch message\n", "22000"},
	Isc_batch_too_big: {"Internal buffer overflow - batch too big\n", "54000"},
	Isc_num_literal: {"Numeric literal too long\n", "54000"},
	Isc_map_event: {"Error using events in mapping shared memory: @1\n", "XX000"},
	Isc_map_overflow: {"Global mapping memory overflow\n", "54000"},
	Isc_hdr_overflow: {"Header page overflow - too many clumplets on it\n", "54000"},
	Isc_vld_plugins: {"No matching client/server authentication plugins configured for execute statement in embedded datasource\n", "28000"},
	Isc_db_crypt_key: {"Missing database encryption key for your attachment\n", "08004"},
	Isc_no_keyholder_plugin: {"Key holder plugin @1 failed to load\n", "HY024"},
	Isc_ses_reset_err: {"Cannot reset user session\n", "01002"},
	Isc_ses_reset_open_trans: {"There are open transactions (@1 active)\n", "25S02"},
	Isc_ses_reset_warn: {"Session was reset with warning(s)\n", "01000"},
	Isc_ses_reset_tran_rollback: {"Transaction is rolled back due to session reset, all changes are lost\n", "01102"},
	Isc_plugin_name: {"Plugin @1:\n", "00000"},
	Isc_parameter_name: {"PARAMETER @1\n", "42000"},
	Isc_file_starting_page_err: {"Starting page number for file @1 must be @2 or greater\n", "HY000"},
	Isc_invalid_timezone_offset: {"Invalid time zone offset: @1 - must use format +/-hours:minutes and be between -14:00 and +14:00\n", "22009"},
	Isc_invalid_timezone_region: {"Invalid time zone region: @1\n", "22009"},
	Isc_invalid_timezone_id: {"Invalid time zone ID: @1\n", "22009"},
	Isc_tom_decode64len: {"Wrong base64 text length @1, should be multiple of 4\n", "22023"},
	Isc_tom_strblob: {"Invalid first parameter datatype - need string or blob\n", "22023"},
	Isc_tom_reg: {"Error registering @1 - probably bad tomcrypt library\n", "22023"},
	Isc_tom_algorithm: {"Unknown crypt algorithm @1 in USING clause\n", "22023"},
	Isc_tom_mode_miss: {"Should specify mode parameter for symmetric cipher\n", "22023"},
	Isc_tom_mode_bad: {"Unknown symmetric crypt mode specified\n", "22023"},
	Isc_tom_no_mode: {"Mode parameter makes no sense for chosen cipher\n", "22023"},
	Isc_tom_iv_miss: {"Should specify initialization vector (IV) for chosen cipher and/or mode\n", "22023"},
	Isc_tom_no_iv: {"Initialization vector (IV) makes no sense for chosen cipher and/or mode\n", "22023"},
	Isc_tom_ctrtype_bad: {"Invalid counter endianess @1\n", "22023"},
	Isc_tom_no_ctrtype: {"Counter endianess parameter is not used in mode @1\n", "22023"},
	Isc_tom_ctr_big: {"Too big counter value @1, maximum @2 can be used\n", "22023"},
	Isc_tom_no_ctr: {"Counter length/value parameter is not used with @1 @2\n", "22023"},
	Isc_tom_iv_length: {"Invalid initialization vector (IV) length @1, need @2\n", "22023"},
	Isc_tom_error: {"TomCrypt library error: @1\n", "22000"},
	Isc_tom_yarrow_start: {"Starting PRNG yarrow\n", "22023"},
	Isc_tom_yarrow_setup: {"Setting up PRNG yarrow\n", "22023"},
	Isc_tom_init_mode: {"Initializing @1 mode\n", "22023"},
	Isc_tom_crypt_mode: {"Encrypting in @1 mode\n", "22023"},
	Isc_tom_decrypt_mode: {"Decrypting in @1 mode\n", "22023"},
	Isc_tom_init_cip: {"Initializing cipher @1\n", "22023"},
	Isc_tom_crypt_cip: {"Encrypting using cipher @1\n", "22023"},
	Isc_tom_decrypt_cip: {"Decrypting using cipher @1\n", "22023"},
	Isc_tom_setup_cip: {"Setting initialization vector (IV) for @1\n", "22023"},
	Isc_tom_setup_chacha: {"Invalid initialization vector (IV) length @1, need  8 or 12\n", "22023"},
	Isc_tom_encode: {"Encoding @1\n", "22023"},
	Isc_tom_decode: {"Decoding @1\n", "22023"},
	Isc_tom_rsa_import: {"Importing RSA key\n", "22023"},
	Isc_tom_oaep: {"Invalid OAEP packet\n", "22023"},
	Isc_tom_hash_bad: {"Unknown hash algorithm @1\n", "22023"},
	Isc_tom_rsa_make: {"Making RSA key\n", "22023"},
	Isc_tom_rsa_export: {"Exporting @1 RSA key\n", "22023"},
	Isc_tom_rsa_sign: {"RSA-signing data\n", "22023"},
	Isc_tom_rsa_verify: {"Verifying RSA-signed data\n", "22023"},
	Isc_tom_chacha_key: {"Invalid key length @1, need 16 or 32\n", "22023"},
	Isc_bad_repl_handle: {"invalid replicator handle\n", "08003"},
	Isc_tra_snapshot_does_not_exist: {"Transaction's base snapshot number does not exist\n", "0B000"},
	Isc_eds_input_prm_not_used: {"Input parameter '@1' is not used in SQL query text\n", "42000"},
	Isc_effective_user: {"Effective user is @1\n", "28000"},
	Isc_invalid_time_zone_bind: {"Invalid time zone bind mode @1\n", "42000"},
	Isc_invalid_decfloat_bind: {"Invalid decfloat bind mode @1\n", "42000"},
	Isc_odd_hex_len: {"Invalid hex text length @1, should be multiple of 2\n", "22023"},
	Isc_invalid_hex_digit: {"Invalid hex digit @1 at position @2\n", "22023"},
	Isc_bind_err: {"Error processing isc_dpb_set_bind clumplet \"@1\"\n", "08004"},
	Isc_bind_statement: {"The following statement failed: @1\n", "08004"},
	Isc_bind_convert: {"Can not convert @1 to @2\n", "22000"},
	Isc_cannot_update_old_blob: {"cannot update old BLOB\n", "42000"},
	Isc_cannot_read_new_blob: {"cannot read from new BLOB\n", "42000"},
	Isc_dyn_no_create_priv: {"No permission for CREATE @1 operation\n", "42000"},
	Isc_suspend_without_returns: {"SUSPEND could not be used without RETURNS clause in PROCEDURE or EXECUTE BLOCK\n", "42000"},
	Isc_truncate_warn: {"String truncated warning due to the following reason\n", "01004"},
	Isc_truncate_monitor: {"Monitoring data does not fit into the field\n", "01004"},
	Isc_truncate_context: {"Engine data does not fit into return value of system function\n", "01004"},
	Isc_merge_dup_update: {"Multiple source records cannot match the same target during MERGE\n", "21000"},
	Isc_wrong_page: {"RDB$PAGES written by non-system transaction, DB appears to be damaged\n", "XX000"},
	Isc_repl_error: {"Replication error\n", "HY000"},
	Isc_ses_reset_failed: {"Reset of user session failed. Connection is shut down.\n", "08003"},
	Isc_block_size: {"File size is less than expected\n", "XX000"},
	Isc_tom_key_length: {"Invalid key length @1, need >@2\n", "22023"},
	Isc_inf_invalid_args: {"Invalid information arguments\n", "HY000"},
	Isc_sysf_invalid_null_empty: {"Empty or NULL parameter @1 is not accepted\n", "22023"},
	Isc_bad_loctab_num: {"Undefined local table number @1\n", "HY000"},
	Isc_quoted_str_bad: {"Invalid text <@1> after quoted string\n", "22024"},
	Isc_quoted_str_miss: {"Missing terminating quote <@1> in the end of quoted string\n", "22024"},
	Isc_wrong_shmem_ver: {"@1: inconsistent shared memory type/version; found @2, expected @3\n", "08006"},
	Isc_wrong_shmem_bitness: {"@1-bit engine can't open database already opened by @2-bit engine\n", "08006"},
	Isc_wrong_proc_plan: {"Procedures cannot specify access type other than NATURAL in the plan\n", "HY000"},
	Isc_invalid_blob_util_handle: {"Invalid RDB$BLOB_UTIL handle\n", "42000"},
	Isc_bad_temp_blob_id: {"Invalid temporary BLOB ID\n", "42000"},
	Isc_ods_upgrade_err: {"ODS upgrade failed while adding new system %s\n", "HY000"},
	Isc_bad_par_workers: {"Wrong parallel workers value @1, valid range are from 1 to @2\n", "HY000"},
	Isc_idx_expr_not_found: {"Definition of index expression is not found for index @1\n", "42000"},
	Isc_idx_cond_not_found: {"Definition of index condition is not found for index @1\n", "42000"},
	Isc_uninitialized_var: {"Variable @1 is not initialized\n", "42000"},
	Isc_param_not_exist: {"Parameter @1 does not exist\n", "07001"},
	Isc_param_no_default_not_specified: {"Parameter @1 has no default value and was not specified or was specified with DEFAULT\n", "07001"},
	Isc_param_multiple_assignments: {"Parameter @1 has multiple assignments\n", "07001"},
	Isc_invalid_date_format: {"Cannot recognize \"@1\" part of date format\n", "HY000"},
	Isc_invalid_raw_string_in_date_format: {"Cannot find closing \" for raw text in date format\n", "HY000"},
	Isc_invalid_data_type_for_date_format: {"It is not possible to use this data type for date formatting\n", "HY000"},
	Isc_incompatible_date_format_with_current_date_type: {"Cannot use \"@1\" format with current date type\n", "HY000"},
	Isc_value_for_pattern_is_out_of_range: {"Value for @1 pattern is out of range [@2, @3]\n", "HY000"},
	Isc_month_name_mismatch: {"@1 is not MONTH\n", "HY000"},
	Isc_incorrect_hours_period: {"@1 is incorrect period for 12H, it should be A.M. or P.M.\n", "HY000"},
	Isc_data_for_format_is_exhausted: {"All data has been read, but format pattern wants more. Unfilled patterns: \"@1\"\n", "HY000"},
	Isc_trailing_part_of_string: {"There is a trailing part of input string that does not fit into FORMAT: \"@1\"\n", "HY000"},
	Isc_pattern_cant_be_used_without_other_pattern: {"@1 can't be used without @2\n", "HY000"},
	Isc_pattern_cant_be_used_without_other_pattern_and_vice_versa: {"@1 can't be used without @2 and vice versa\n", "HY000"},
	Isc_incompatible_format_patterns: {"@1 incompatible with @2\n", "HY000"},
	Isc_only_one_pattern_can_be_used: {"Can use only one of these patterns @1\n", "HY000"},
	Isc_can_not_use_same_pattern_twice: {"Cannot use the same pattern twice: @1\n", "HY000"},
	Isc_sysf_invalid_gen_uuid_version: {"Invalid GEN_UUID version (@1). Must be 4 or 7\n", "42000"},
	Isc_sweep_unable_to_run: {"Unable to run sweep\n", "42000"},
	Isc_sweep_concurrent_instance: {"Another instance of sweep is already running\n", "42000"},
	Isc_sweep_read_only: {"Database in read only state\n", "42000"},
	Isc_sweep_attach_no_cleanup: {"Attachment has no cleanup flag set\n", "42000"},
	Isc_invalid_timezone_region_or_displacement: {"Invalid time zone region or displacement: @1\n", "HY000"},
	Isc_argmustbe_exact_range_for: {"Arguments for range-based FOR must be exact numeric types\n", "42000"},
	Isc_range_for_by_should_be_positive: {"Range-based FOR BY argument must be positive\n", "42000"},
	Isc_missing_value_for_format_pattern: {"Cannot find value in input string for \"@1\" pattern\n", "HY000"},
	Isc_invalid_name: {"Invalid name: @1\n", "HY000"},
	Isc_invalid_unqualified_name_list: {"Invalid list of unqualified names: @1\n", "HY000"},
	Isc_no_user_att_while_restore: {"User attachments are not allowed for the database being restored\n", "HY000"},
	Isc_genseq_stepmustbe_nonzero: {"Argument STEP must be different than zero for function @1\n", "42000"},
	Isc_argmustbe_exact_function: {"Arguments for @1 function must be exact numeric types\n", "42000"},
	Isc_bad_ext_file: {"Invalid external file format\n", "22000"},
	Isc_wrong_adp_fields_def: {"Wrong set of adapter fields\n", "42000"},
	Isc_wrong_adp_field_name: {"Adapter field named \"@1\" is not supported\n", "42000"},
	Isc_wrong_adp_field_type: {"Wrong type of adapter field \"@1\"\n", "42000"},
	Isc_bad_ext_record: {"Unknown record type in external file \"@1\" at offset @2\n", "22000"},
	Isc_bad_adp_type: {"Unknown adapter type \"@1\"\n", "42000"},
	Isc_dir_resolve_error: {"Cannot resolve directory name \"@1\"\n", "HY000"},
	Isc_read_file_error: {"Error reading file \"@1\": @2\n", "HY000"},
	Isc_create_file_error: {"Error creating file \"@1\": @2\n", "HY000"},
	Isc_delete_file_error: {"Error deleting file \"@1\": @2\n", "HY000"},
	Isc_blob_io_error: {"I/O error during \"@1\" operation for BLOB file \"@2\"\n", "42000"},
	Isc_cant_connect_to_ldap: {"Cannot connect to LDAP server: @1\n", "28000"},
	Isc_cant_bind_to_ldap: {"Cannot bind to LDAP server with specified login and password\n", "28000"},
	Isc_no_ldap_init: {"Cannot initialize LDAP plugin\n", "28000"},
	Isc_sysf_ldap_attr: {"Cannot get user attribute \"@1\" from LDAP\n", "42000"},
	Isc_no_ext_file: {"Missing external file for adapter\n", "22000"},
	Isc_no_gss_init: {"GSS library was not loaded correctly\n", "HY000"},
	Isc_hash_error: {"Hash calculation error\n", "HY000"},
	Isc_policy_already_exists: {"Policy @1 already exists\n", "22028"},
	Isc_policy_not_exists: {"Policy @1 does not exist\n", "02000"},
	Isc_user_not_exists: {"User @1 does not exist\n", "02000"},
	Isc_policy_is_granted: {"Policy @1 is granted to a user\n", "23000"},
	Isc_bad_trig_BLR: {"Error while parsing trigger @1's BLR\n", "2F000"},
	Isc_not_rdb: {"Unable to open database @1, is not Red Database\n", "HY000"},
	Isc_trace_write_error: {"Error when trying to write to the trace file\n", "22000"},
	Isc_no_granted_policy: {"User @1 does not have granted policy\n", "02000"},
	Isc_regexp_substr_error: {"Error calculating substring using regular expression: @2\n", "42000"},
	Isc_ts_file_exists: {"Tablespace \"@1\" creation error. File \"@2\" exists.\n", "08001"},
	Isc_tablespace_name: {"TABLESPACE @1\n", "42000"},
	Isc_ts_file_not_exists: {"Tablespace \"@1\" alteration error. File \"@2\" does not exist.\n", "08001"},
	Isc_secdb_coonection_error: {"Error while trying to attach the security DB\n", "22000"},
	Isc_policies_reading_error: {"Error while trying to read the PLG$POLICIES\n", "22000"},
	Isc_get_policies_error: {"@1 failed during selecting from SEC$POLICIES\n", "22000"},
	Isc_att_suspended: {"Connection suspended\n", "22000"},
	Isc_suspend_privileges_error: {"not enough privileges to complete operation\n", "22000"},
	Isc_trace_buffer_limit: {"Size of trace message buffer (@1) exceeds limit @2\n", "HY000"},
	Isc_pman_hash_incorrect: {"Hash is incorrect\n", "00000"},
	Isc_json_table_plan_missing_name: {"The nested column name @1 does not appear in the plan clause\n", "22000"},
	Isc_json_table_same_names: {"The column name @1 belongs to several columns\n", "22000"},
	Isc_json_table_plan_same_names: {"The path name @1 belongs to several columns\n", "22000"},
	Isc_json_table_plan_double_appearance: {"All the path names (including @1) must appear in the PLAN clause exactly once\n", "22000"},
	Isc_json_table_plan_wrong_level: {"The nested @1 is on the wrong level ib the plan\n", "22000"},
	Isc_json_table_invalid_plan: {"The json table plan is invalid\n", "22000"},
	Isc_json_table_plan_wrong_name: {"There is no nested column name with named @1\n", "22000"},
	Isc_foreign_server: {"foreign server @1\n", "42000"},
	Isc_foreign_provider_not_found: {"Foreign provider '@1' for server '@2' not found\n", "42000"},
	Isc_change_forced_write_mode_with_connections: {"Can't change forced write mode while server has connections\n", "42000"},
	Isc_license_not_found: {"License plugin is not found\n", "HY000"},
	Isc_feature_not_available: {"@1 is not available in this edition\n", "HY000"},
	Isc_repl_mispos: {"Replication packet is mispositioned. Replica position is (@1:@2)\n", "HY000"},
	Isc_missing_trace_plugin: {"The '@1' plugin is missing. Please specify it in the TracePlugin option and restart the server\n", "HY000"},
	Isc_repl_recover_err: {"Database recovery failed\n", "HY000"},
	Isc_user_policy_reading_error: {"Error while trying to read the PLG$USER_POLICY\n", "22000"},
	Isc_get_user_policy_error: {"@1 failed during selecting from SEC$USER_POLICY\n", "22000"},
	Isc_gfix_db_name: {"data base file name (@1) already given\n", "00000"},
	Isc_gfix_invalid_sw: {"invalid switch @1\n", "00000"},
	Isc_gfix_incmp_sw: {"incompatible switch combination\n", "00000"},
	Isc_gfix_replay_req: {"replay log pathname required\n", "00000"},
	Isc_gfix_pgbuf_req: {"number of page buffers for cache required\n", "00000"},
	Isc_gfix_val_req: {"numeric value required\n", "00000"},
	Isc_gfix_pval_req: {"positive numeric value required\n", "00000"},
	Isc_gfix_trn_req: {"number of transactions per sweep required\n", "00000"},
	Isc_gfix_full_req: {"\"full\" or \"reserve\" required\n", "00000"},
	Isc_gfix_usrname_req: {"user name required\n", "00000"},
	Isc_gfix_pass_req: {"password required\n", "00000"},
	Isc_gfix_subs_name: {"subsystem name\n", "00000"},
	Isc_gfix_wal_req: {"\"wal\" required\n", "00000"},
	Isc_gfix_sec_req: {"number of seconds required\n", "00000"},
	Isc_gfix_nval_req: {"numeric value between 0 and 32767 inclusive required\n", "00000"},
	Isc_gfix_type_shut: {"must specify type of shutdown\n", "00000"},
	Isc_gfix_retry: {"please retry, specifying an option\n", "00000"},
	Isc_gfix_retry_db: {"please retry, giving a database name\n", "00000"},
	Isc_gfix_exceed_max: {"internal block exceeds maximum size\n", "00000"},
	Isc_gfix_corrupt_pool: {"corrupt pool\n", "00000"},
	Isc_gfix_mem_exhausted: {"virtual memory exhausted\n", "00000"},
	Isc_gfix_bad_pool: {"bad pool id\n", "00000"},
	Isc_gfix_trn_not_valid: {"Transaction state @1 not in valid range.\n", "00000"},
	Isc_gfix_unexp_eoi: {"unexpected end of input\n", "00000"},
	Isc_gfix_recon_fail: {"failed to reconnect to a transaction in database @1\n", "00000"},
	Isc_gfix_trn_unknown: {"Transaction description item unknown\n", "00000"},
	Isc_gfix_mode_req: {"\"read_only\" or \"read_write\" required\n", "00000"},
	Isc_gfix_pzval_req: {"positive or zero numeric value required\n", "00000"},
	Isc_gfix_overval_req: {"too big numeric value\n", "00000"},
	Isc_dsql_dbkey_from_non_table: {"Cannot SELECT RDB$DB_KEY from a stored procedure.\n", "HY000"},
	Isc_dsql_transitional_numeric: {"Precision 10 to 18 changed from DOUBLE PRECISION in SQL dialect 1 to 64-bit scaled integer in SQL dialect 3\n", "HY000"},
	Isc_dsql_dialect_warning_expr: {"Use of @1 expression that returns different results in dialect 1 and dialect 3\n", "01000"},
	Isc_sql_db_dialect_dtype_unsupport: {"Database SQL dialect @1 does not support reference to @2 datatype\n", "HY000"},
	Isc_sql_dialect_conflict_num: {"DB dialect @1 and client dialect @2 conflict with respect to numeric precision @3.\n", "HY000"},
	Isc_dsql_warning_number_ambiguous: {"WARNING: Numeric literal @1 is interpreted as a floating-point\n", "HY104"},
	Isc_dsql_warning_number_ambiguous1: {"value in SQL dialect 1, but as an exact numeric value in SQL dialect 3.\n", "HY104"},
	Isc_dsql_warn_precision_ambiguous: {"WARNING: NUMERIC and DECIMAL fields with precision 10 or greater are stored\n", "HY104"},
	Isc_dsql_warn_precision_ambiguous1: {"as approximate floating-point values in SQL dialect 1, but as 64-bit\n", "HY104"},
	Isc_dsql_warn_precision_ambiguous2: {"integers in SQL dialect 3.\n", "HY104"},
	Isc_dsql_ambiguous_field_name: {"Ambiguous field name between @1 and @2\n", "42702"},
	Isc_dsql_udf_return_pos_err: {"External function should have return position between 1 and @1\n", "38000"},
	Isc_dsql_invalid_label: {"Label @1 @2 in the current scope\n", "HY000"},
	Isc_dsql_datatypes_not_comparable: {"Datatypes @1are not comparable in expression @2\n", "HY004"},
	Isc_dsql_cursor_invalid: {"Empty cursor name is not allowed\n", "24000"},
	Isc_dsql_cursor_redefined: {"Statement already has a cursor @1 assigned\n", "24000"},
	Isc_dsql_cursor_not_found: {"Cursor @1 is not found in the current context\n", "34000"},
	Isc_dsql_cursor_exists: {"Cursor @1 already exists in the current context\n", "24000"},
	Isc_dsql_cursor_rel_ambiguous: {"Relation @1 is ambiguous in cursor @2\n", "34000"},
	Isc_dsql_cursor_rel_not_found: {"Relation @1 is not found in cursor @2\n", "34000"},
	Isc_dsql_cursor_not_open: {"Cursor is not open\n", "24000"},
	Isc_dsql_type_not_supp_ext_tab: {"Data type @1 is not supported for EXTERNAL TABLES. Relation @2, field @3\n", "HY004"},
	Isc_dsql_feature_not_supported_ods: {"Feature not supported on ODS version older than @1.@2\n", "0A000"},
	Isc_primary_key_required: {"Primary key required on table @1\n", "22000"},
	Isc_upd_ins_doesnt_match_pk: {"UPDATE OR INSERT field list does not match primary key of table @1\n", "42000"},
	Isc_upd_ins_doesnt_match_matching: {"UPDATE OR INSERT field list does not match MATCHING clause\n", "42000"},
	Isc_upd_ins_with_complex_view: {"UPDATE OR INSERT without MATCHING could not be used with views based on more than one table\n", "54001"},
	Isc_dsql_incompatible_trigger_type: {"Incompatible trigger type\n", "42000"},
	Isc_dsql_db_trigger_type_cant_change: {"Database trigger type can't be changed\n", "42000"},
	Isc_dsql_record_version_table: {"To be used with RDB$RECORD_VERSION, @1 must be a table or a view of single table\n", "HY000"},
	Isc_dsql_invalid_sqlda_version: {"SQLDA version expected between @1 and @2, found @3\n", "07002"},
	Isc_dsql_sqlvar_index: {"at SQLVAR index @1\n", "07002"},
	Isc_dsql_no_sqlind: {"empty pointer to NULL indicator variable\n", "07002"},
	Isc_dsql_no_sqldata: {"empty pointer to data\n", "07002"},
	Isc_dsql_no_input_sqlda: {"No SQLDA for input values provided\n", "07002"},
	Isc_dsql_no_output_sqlda: {"No SQLDA for output values provided\n", "07002"},
	Isc_dsql_wrong_param_num: {"Wrong number of parameters (expected @1, got @2)\n", "07001"},
	Isc_dsql_invalid_drop_ss_clause: {"Invalid DROP SQL SECURITY clause\n", "42000"},
	Isc_upd_ins_cannot_default: {"UPDATE OR INSERT value for field @1, part of the implicit or explicit MATCHING clause, cannot be DEFAULT\n", "42000"},
	Isc_dyn_filter_not_found: {"BLOB Filter @1 not found\n", "42000"},
	Isc_dyn_func_not_found: {"Function @1 not found\n", "42000"},
	Isc_dyn_index_not_found: {"Index not found\n", "42000"},
	Isc_dyn_view_not_found: {"View @1 not found\n", "42000"},
	Isc_dyn_domain_not_found: {"Domain not found\n", "42000"},
	Isc_dyn_cant_modify_auto_trig: {"Triggers created automatically cannot be modified\n", "42000"},
	Isc_dyn_dup_table: {"Table @1 already exists\n", "42S01"},
	Isc_dyn_proc_not_found: {"Procedure @1 not found\n", "42000"},
	Isc_dyn_exception_not_found: {"Exception not found\n", "42000"},
	Isc_dyn_proc_param_not_found: {"Parameter @1 in procedure @2 not found\n", "42000"},
	Isc_dyn_trig_not_found: {"Trigger @1 not found\n", "42000"},
	Isc_dyn_charset_not_found: {"Character set @1 not found\n", "42000"},
	Isc_dyn_collation_not_found: {"Collation @1 not found\n", "42000"},
	Isc_dyn_role_not_found: {"Role @1 not found\n", "42000"},
	Isc_dyn_name_longer: {"Name longer than database column size\n", "42000"},
	Isc_dyn_column_does_not_exist: {"column @1 does not exist in table/view @2\n", "42S22"},
	Isc_dyn_role_does_not_exist: {"SQL role @1 does not exist\n", "28000"},
	Isc_dyn_no_grant_admin_opt: {"user @1 has no grant admin option on SQL role @2\n", "28000"},
	Isc_dyn_user_not_role_member: {"user @1 is not a member of SQL role @2\n", "28000"},
	Isc_dyn_delete_role_failed: {"@1 is not the owner of SQL role @2\n", "28000"},
	Isc_dyn_grant_role_to_user: {"@1 is a SQL role and not a user\n", "28000"},
	Isc_dyn_inv_sql_role_name: {"user name @1 could not be used for SQL role\n", "28000"},
	Isc_dyn_dup_sql_role: {"SQL role @1 already exists\n", "42000"},
	Isc_dyn_kywd_spec_for_role: {"keyword @1 can not be used as a SQL role name\n", "28000"},
	Isc_dyn_roles_not_supported: {"SQL roles are not supported in on older versions of the database.  A backup and restore of the database is required.\n", "28000"},
	Isc_dyn_domain_name_exists: {"Cannot rename domain @1 to @2.  A domain with that name already exists.\n", "42000"},
	Isc_dyn_field_name_exists: {"Cannot rename column @1 to @2.  A column with that name already exists in table @3.\n", "42S21"},
	Isc_dyn_dependency_exists: {"Column @1 from table @2 is referenced in @3\n", "42000"},
	Isc_dyn_dtype_invalid: {"Cannot change datatype for column @1.  Changing datatype is not supported for BLOB or ARRAY columns.\n", "42000"},
	Isc_dyn_char_fld_too_small: {"New size specified for column @1 must be at least @2 characters.\n", "42000"},
	Isc_dyn_invalid_dtype_conversion: {"Cannot change datatype for @1.  Conversion from base type @2 to @3 is not supported.\n", "42000"},
	Isc_dyn_dtype_conv_invalid: {"Cannot change datatype for column @1 from a character type to a non-character type.\n", "42000"},
	Isc_dyn_zero_len_id: {"Zero length identifiers are not allowed\n", "42000"},
	Isc_dyn_gen_not_found: {"Sequence @1 not found\n", "42000"},
	Isc_max_coll_per_charset: {"Maximum number of collations per character set exceeded\n", "2C000"},
	Isc_invalid_coll_attr: {"Invalid collation attributes\n", "HY000"},
	Isc_dyn_wrong_gtt_scope: {"@1 cannot reference @2\n", "HY000"},
	Isc_dyn_coll_used_table: {"Collation @1 is used in table @2 (field name @3) and cannot be dropped\n", "HY000"},
	Isc_dyn_coll_used_domain: {"Collation @1 is used in domain @2 and cannot be dropped\n", "HY000"},
	Isc_dyn_cannot_del_syscoll: {"Cannot delete system collation\n", "HY000"},
	Isc_dyn_cannot_del_def_coll: {"Cannot delete default collation of CHARACTER SET @1\n", "HY000"},
	Isc_dyn_table_not_found: {"Table @1 not found\n", "42000"},
	Isc_dyn_coll_used_procedure: {"Collation @1 is used in procedure @2 (parameter name @3) and cannot be dropped\n", "HY000"},
	Isc_dyn_scale_too_big: {"New scale specified for column @1 must be at most @2.\n", "42000"},
	Isc_dyn_precision_too_small: {"New precision specified for column @1 must be at least @2.\n", "42000"},
	Isc_dyn_miss_priv_warning: {"Warning: @1 on @2 is not granted to @3.\n", "42000"},
	Isc_dyn_ods_not_supp_feature: {"Feature '@1' is not supported in ODS @2.@3\n", "0A000"},
	Isc_dyn_cannot_addrem_computed: {"Cannot add or remove COMPUTED from column @1\n", "42000"},
	Isc_dyn_no_empty_pw: {"Password should not be empty string\n", "42000"},
	Isc_dyn_dup_index: {"Index @1 already exists\n", "42S11"},
	Isc_dyn_package_not_found: {"Package @1 not found\n", "42000"},
	Isc_dyn_schema_not_found: {"Schema @1 not found\n", "42000"},
	Isc_dyn_cannot_mod_sysproc: {"Cannot ALTER or DROP system procedure @1\n", "HY000"},
	Isc_dyn_cannot_mod_systrig: {"Cannot ALTER or DROP system trigger @1\n", "HY000"},
	Isc_dyn_cannot_mod_sysfunc: {"Cannot ALTER or DROP system function @1\n", "HY000"},
	Isc_dyn_invalid_ddl_proc: {"Invalid DDL statement for procedure @1\n", "HY000"},
	Isc_dyn_invalid_ddl_trig: {"Invalid DDL statement for trigger @1\n", "HY000"},
	Isc_dyn_funcnotdef_package: {"Function @1 has not been defined on the package body @2\n", "42000"},
	Isc_dyn_procnotdef_package: {"Procedure @1 has not been defined on the package body @2\n", "42000"},
	Isc_dyn_funcsignat_package: {"Function @1 has a signature mismatch on package body @2\n", "42000"},
	Isc_dyn_procsignat_package: {"Procedure @1 has a signature mismatch on package body @2\n", "42000"},
	Isc_dyn_defvaldecl_package_proc: {"Default values for parameters are not allowed in the definition of a previously declared packaged procedure @1.@2\n", "42000"},
	Isc_dyn_package_body_exists: {"Package body @1 already exists\n", "42000"},
	Isc_dyn_invalid_ddl_func: {"Invalid DDL statement for function @1\n", "HY000"},
	Isc_dyn_newfc_oldsyntax: {"Cannot alter new style function @1 with ALTER EXTERNAL FUNCTION. Use ALTER FUNCTION instead.\n", "42000"},
	Isc_dyn_func_param_not_found: {"Parameter @1 in function @2 not found\n", "42000"},
	Isc_dyn_routine_param_not_found: {"Parameter @1 of routine @2 not found\n", "42000"},
	Isc_dyn_routine_param_ambiguous: {"Parameter @1 of routine @2 is ambiguous (found in both procedures and functions). Use a specifier keyword.\n", "42000"},
	Isc_dyn_coll_used_function: {"Collation @1 is used in function @2 (parameter name @3) and cannot be dropped\n", "HY000"},
	Isc_dyn_domain_used_function: {"Domain @1 is used in function @2 (parameter name @3) and cannot be dropped\n", "HY000"},
	Isc_dyn_alter_user_no_clause: {"ALTER USER requires at least one clause to be specified\n", "42000"},
	Isc_dyn_duplicate_package_item: {"Duplicate @1 @2\n", "42000"},
	Isc_dyn_cant_modify_sysobj: {"System @1 @2 cannot be modified\n", "42000"},
	Isc_dyn_cant_use_zero_increment: {"INCREMENT BY 0 is an illegal option for sequence @1\n", "42000"},
	Isc_dyn_cant_use_in_foreignkey: {"Can't use @1 in FOREIGN KEY constraint\n", "42000"},
	Isc_dyn_defvaldecl_package_func: {"Default values for parameters are not allowed in the definition of a previously declared packaged function @1.@2\n", "42000"},
	Isc_dyn_cyclic_role: {"role @1 can not be granted to role @2\n", "42000"},
	Isc_dyn_cant_use_zero_inc_ident: {"INCREMENT BY 0 is an illegal option for identity column @1 of table @2\n", "42000"},
	Isc_dyn_no_ddl_grant_opt_priv: {"no @1 privilege with grant option on DDL @2\n", "42000"},
	Isc_dyn_no_grant_opt_priv: {"no @1 privilege with grant option on object @2\n", "42000"},
	Isc_dyn_func_not_exist: {"Function @1 does not exist\n", "42000"},
	Isc_dyn_proc_not_exist: {"Procedure @1 does not exist\n", "42000"},
	Isc_dyn_pack_not_exist: {"Package @1 does not exist\n", "42000"},
	Isc_dyn_trig_not_exist: {"Trigger @1 does not exist\n", "42000"},
	Isc_dyn_view_not_exist: {"View @1 does not exist\n", "42000"},
	Isc_dyn_rel_not_exist: {"Table @1 does not exist\n", "42000"},
	Isc_dyn_exc_not_exist: {"Exception @1 does not exist\n", "42000"},
	Isc_dyn_gen_not_exist: {"Generator/Sequence @1 does not exist\n", "42000"},
	Isc_dyn_fld_not_exist: {"Field @1 of table @2 does not exist\n", "42000"},
	Isc_dyn_index_schema_must_match_table: {"Index schema (@1) must match table schema (@2)\n", "42000"},
	Isc_dyn_trig_schema_must_match_table: {"Trigger schema (@1) must match table schema (@2)\n", "42000"},
	Isc_dyn_cannot_mod_system_schema: {"Cannot ALTER or DROP SYSTEM schema\n", "28000"},
	Isc_dyn_cannot_drop_non_emptyschema: {"Cannot DROP schema @1 because it has objects\n", "HY000"},
	Isc_dyn_cannot_mod_obj_sys_schema: {"Cannot CREATE/ALTER/DROP @1 in SYSTEM schema\n", "28000"},
	Isc_dyn_cannot_create_reserved_schema: {"Schema name @1 is reserved and cannot be created\n", "HY000"},
	Isc_dyn_cannot_infer_schema: {"Cannot infer schema name as there is no valid schema in the search path\n", "42000"},
	Isc_dyn_dup_domain_rdb: {"Domain @1 already exists\n", "42000"},
	Isc_dyn_dup_trigger_rdb: {"Trigger @1 already exists\n", "42000"},
	Isc_dyn_dup_package_rdb: {"Package @1 already exists\n", "42000"},
	Isc_dyn_dup_filter: {"Filter @1 already exists\n", "42000"},
	Isc_dyn_job_not_found: {"Job @1 not found\n", "42000"},
	Isc_dyn_invalid_ddl_job: {"Invalid DDL statement for job @1\n", "HY000"},
	Isc_dyn_alter_job_no_clause: {"ALTER JOB requires at least one clause to be specified\n", "42000"},
	Isc_dyn_ts_not_found: {"Tablespace @1 not found\n", "42000"},
	Isc_dyn_cant_set_ts_table: {"Cannot set tablespace for temporary table @1\n", "42000"},
	Isc_dyn_cant_modify_collation: {"Cannot change collation of field '@1.@2' to @3\n", "42000"},
	Isc_dyn_cant_set_ts_index: {"Cannot set tablespace for temporary index @1\n", "42000"},
	Isc_dyn_foreign_server_not_found: {"Foreign server @1 not found\n", "42000"},
	Isc_dyn_user_mapping_not_found: {"User mapping for user @1 and server @2 not found\n", "42000"},
	Isc_dyn_job_already_exists: {"Job @1 already exists\n", "42000"},
	Isc_dyn_ts_already_exists: {"Tablespace @1 already exists\n", "42000"},
	Isc_dyn_foreign_server_already_exists: {"Foreign server @1 already exists\n", "42000"},
	Isc_dyn_cant_set_ts_external_table: {"Cannot set tablespace for external table @1\n", "42000"},
	Isc_gbak_unknown_switch: {"found unknown switch\n", "00000"},
	Isc_gbak_page_size_missing: {"page size parameter missing\n", "00000"},
	Isc_gbak_page_size_toobig: {"Page size specified (@1) greater than limit (32768 bytes)\n", "00000"},
	Isc_gbak_redir_ouput_missing: {"redirect location for output is not specified\n", "00000"},
	Isc_gbak_switches_conflict: {"conflicting switches for backup/restore\n", "00000"},
	Isc_gbak_unknown_device: {"device type @1 not known\n", "00000"},
	Isc_gbak_no_protection: {"protection is not there yet\n", "00000"},
	Isc_gbak_page_size_not_allowed: {"page size is allowed only on restore or create\n", "00000"},
	Isc_gbak_multi_source_dest: {"multiple sources or destinations specified\n", "00000"},
	Isc_gbak_filename_missing: {"requires both input and output filenames\n", "00000"},
	Isc_gbak_dup_inout_names: {"input and output have the same name.  Disallowed.\n", "00000"},
	Isc_gbak_inv_page_size: {"expected page size, encountered \"@1\"\n", "00000"},
	Isc_gbak_db_specified: {"REPLACE specified, but the first file @1 is a database\n", "00000"},
	Isc_gbak_db_exists: {"database @1 already exists.  To replace it, use the -REP switch\n", "00000"},
	Isc_gbak_unk_device: {"device type not specified\n", "00000"},
	Isc_gbak_blob_info_failed: {"gds_$blob_info failed\n", "00000"},
	Isc_gbak_unk_blob_item: {"do not understand BLOB INFO item @1\n", "00000"},
	Isc_gbak_get_seg_failed: {"gds_$get_segment failed\n", "00000"},
	Isc_gbak_close_blob_failed: {"gds_$close_blob failed\n", "00000"},
	Isc_gbak_open_blob_failed: {"gds_$open_blob failed\n", "00000"},
	Isc_gbak_put_blr_gen_id_failed: {"Failed in put_blr_gen_id\n", "00000"},
	Isc_gbak_unk_type: {"data type @1 not understood\n", "00000"},
	Isc_gbak_comp_req_failed: {"gds_$compile_request failed\n", "00000"},
	Isc_gbak_start_req_failed: {"gds_$start_request failed\n", "00000"},
	Isc_gbak_rec_failed: {"gds_$receive failed\n", "00000"},
	Isc_gbak_rel_req_failed: {"gds_$release_request failed\n", "00000"},
	Isc_gbak_db_info_failed: {"gds_$database_info failed\n", "00000"},
	Isc_gbak_no_db_desc: {"Expected database description record\n", "00000"},
	Isc_gbak_db_create_failed: {"failed to create database @1\n", "00000"},
	Isc_gbak_decomp_len_error: {"RESTORE: decompression length error\n", "00000"},
	Isc_gbak_tbl_missing: {"cannot find table @1\n", "00000"},
	Isc_gbak_blob_col_missing: {"Cannot find column for BLOB\n", "00000"},
	Isc_gbak_create_blob_failed: {"gds_$create_blob failed\n", "00000"},
	Isc_gbak_put_seg_failed: {"gds_$put_segment failed\n", "00000"},
	Isc_gbak_rec_len_exp: {"expected record length\n", "00000"},
	Isc_gbak_inv_rec_len: {"wrong length record, expected @1 encountered @2\n", "00000"},
	Isc_gbak_exp_data_type: {"expected data attribute\n", "00000"},
	Isc_gbak_gen_id_failed: {"Failed in store_blr_gen_id\n", "00000"},
	Isc_gbak_unk_rec_type: {"do not recognize record type @1\n", "00000"},
	Isc_gbak_inv_bkup_ver: {"Expected backup version 1..10.  Found @1\n", "00000"},
	Isc_gbak_missing_bkup_desc: {"expected backup description record\n", "00000"},
	Isc_gbak_string_trunc: {"string truncated\n", "00000"},
	Isc_gbak_cant_rest_record: {"warning -- record could not be restored\n", "00000"},
	Isc_gbak_send_failed: {"gds_$send failed\n", "00000"},
	Isc_gbak_no_tbl_name: {"no table name for data\n", "00000"},
	Isc_gbak_unexp_eof: {"unexpected end of file on backup file\n", "00000"},
	Isc_gbak_db_format_too_old: {"database format @1 is too old to restore to\n", "00000"},
	Isc_gbak_inv_array_dim: {"array dimension for column @1 is invalid\n", "00000"},
	Isc_gbak_xdr_len_expected: {"Expected XDR record length\n", "00000"},
	Isc_gbak_open_bkup_error: {"cannot open backup file @1\n", "00000"},
	Isc_gbak_open_error: {"cannot open status and error output file @1\n", "00000"},
	Isc_gbak_missing_block_fac: {"blocking factor parameter missing\n", "00000"},
	Isc_gbak_inv_block_fac: {"expected blocking factor, encountered \"@1\"\n", "00000"},
	Isc_gbak_block_fac_specified: {"a blocking factor may not be used in conjunction with device CT\n", "00000"},
	Isc_gbak_missing_username: {"user name parameter missing\n", "00000"},
	Isc_gbak_missing_password: {"password parameter missing\n", "00000"},
	Isc_gbak_missing_skipped_bytes: {" missing parameter for the number of bytes to be skipped\n", "00000"},
	Isc_gbak_inv_skipped_bytes: {"expected number of bytes to be skipped, encountered \"@1\"\n", "00000"},
	Isc_gbak_err_restore_charset: {"character set\n", "00000"},
	Isc_gbak_err_restore_collation: {"collation\n", "00000"},
	Isc_gbak_read_error: {"Unexpected I/O error while reading from backup file\n", "00000"},
	Isc_gbak_write_error: {"Unexpected I/O error while writing to backup file\n", "00000"},
	Isc_gbak_db_in_use: {"could not drop database @1 (no privilege or database might be in use)\n", "00000"},
	Isc_gbak_sysmemex: {"System memory exhausted\n", "00000"},
	Isc_gbak_restore_role_failed: {"SQL role\n", "00000"},
	Isc_gbak_role_op_missing: {"SQL role parameter missing\n", "00000"},
	Isc_gbak_page_buffers_missing: {"page buffers parameter missing\n", "00000"},
	Isc_gbak_page_buffers_wrong_param: {"expected page buffers, encountered \"@1\"\n", "00000"},
	Isc_gbak_page_buffers_restore: {"page buffers is allowed only on restore or create\n", "00000"},
	Isc_gbak_inv_size: {"size specification either missing or incorrect for file @1\n", "00000"},
	Isc_gbak_file_outof_sequence: {"file @1 out of sequence\n", "00000"},
	Isc_gbak_join_file_missing: {"can't join -- one of the files missing\n", "00000"},
	Isc_gbak_stdin_not_supptd: {" standard input is not supported when using join operation\n", "00000"},
	Isc_gbak_stdout_not_supptd: {"standard output is not supported when using split operation or in verbose mode\n", "00000"},
	Isc_gbak_bkup_corrupt: {"backup file @1 might be corrupt\n", "00000"},
	Isc_gbak_unk_db_file_spec: {"database file specification missing\n", "00000"},
	Isc_gbak_hdr_write_failed: {"can't write a header record to file @1\n", "00000"},
	Isc_gbak_disk_space_ex: {"free disk space exhausted\n", "00000"},
	Isc_gbak_size_lt_min: {"file size given (@1) is less than minimum allowed (@2)\n", "00000"},
	Isc_gbak_svc_name_missing: {"service name parameter missing\n", "00000"},
	Isc_gbak_not_ownr: {"Cannot restore over current database, must be SYSDBA or owner of the existing database.\n", "00000"},
	Isc_gbak_mode_req: {"\"read_only\" or \"read_write\" required\n", "00000"},
	Isc_gbak_just_data: {"just data ignore all constraints etc.\n", "00000"},
	Isc_gbak_data_only: {"restoring data only ignoring foreign key, unique, not null & other constraints\n", "00000"},
	Isc_gbak_missing_interval: {"verbose interval value parameter missing\n", "00000"},
	Isc_gbak_wrong_interval: {"verbose interval value cannot be smaller than @1\n", "00000"},
	Isc_gbak_verify_verbint: {"verify (verbose) and verbint options are mutually exclusive\n", "00000"},
	Isc_gbak_option_only_restore: {"option -@1 is allowed only on restore or create\n", "00000"},
	Isc_gbak_option_only_backup: {"option -@1 is allowed only on backup\n", "00000"},
	Isc_gbak_option_conflict: {"options -@1 and -@2 are mutually exclusive\n", "00000"},
	Isc_gbak_param_conflict: {"parameter for option -@1 was already specified with value \"@2\"\n", "00000"},
	Isc_gbak_option_repeated: {"option -@1 was already specified\n", "00000"},
	Isc_gbak_max_dbkey_recursion: {"dependency depth greater than @1 for view @2\n", "00000"},
	Isc_gbak_max_dbkey_length: {"value greater than @1 when calculating length of rdb$db_key for view @2\n", "00000"},
	Isc_gbak_invalid_metadata: {"Invalid metadata detected. Use -FIX_FSS_METADATA option.\n", "00000"},
	Isc_gbak_invalid_data: {"Invalid data detected. Use -FIX_FSS_DATA option.\n", "00000"},
	Isc_gbak_inv_bkup_ver2: {"Expected backup version @2..@3.  Found @1\n", "00000"},
	Isc_gbak_db_format_too_old2: {"database format @1 is too old to backup\n", "00000"},
	Isc_gbak_db_not_online: {"Database is not online due to failure to restore one or more objects.\n", "00000"},
	Isc_gbak_gfix_online: {"Run gfix -online to bring database online.\n", "00000"},
	Isc_gbak_db_not_online_reason: {"Database is not online due to @1.\n", "00000"},
	Isc_dsql_too_old_ods: {"ODS versions before ODS@1 are not supported\n", "HY000"},
	Isc_dsql_table_not_found: {"Table @1 does not exist\n", "42S02"},
	Isc_dsql_view_not_found: {"View @1 does not exist\n", "42S02"},
	Isc_dsql_line_col_error: {"At line @1, column @2\n", "42000"},
	Isc_dsql_unknown_pos: {"At unknown line and column\n", "42000"},
	Isc_dsql_no_dup_name: {"Column @1 cannot be repeated in @2 statement\n", "HY000"},
	Isc_dsql_too_many_values: {"Too many values (more than @1) in member list to match against\n", "07002"},
	Isc_dsql_no_array_computed: {"Array and BLOB data types not allowed in computed field\n", "HY000"},
	Isc_dsql_implicit_domain_name: {"Implicit domain name @1 not allowed in user created domain\n", "00000"},
	Isc_dsql_only_can_subscript_array: {"scalar operator used on field @1 which is not an array\n", "42000"},
	Isc_dsql_max_sort_items: {"cannot sort on more than 255 items\n", "54011"},
	Isc_dsql_max_group_items: {"cannot group on more than 255 items\n", "54011"},
	Isc_dsql_conflicting_sort_field: {"Cannot include the same field (@1.@2) twice in the ORDER BY clause with conflicting sorting options\n", "42000"},
	Isc_dsql_derived_table_more_columns: {"column list from derived table @1 has more columns than the number of items in its SELECT statement\n", "54001"},
	Isc_dsql_derived_table_less_columns: {"column list from derived table @1 has less columns than the number of items in its SELECT statement\n", "54001"},
	Isc_dsql_derived_field_unnamed: {"no column name specified for column number @1 in derived table @2\n", "42000"},
	Isc_dsql_derived_field_dup_name: {"column @1 was specified multiple times for derived table @2\n", "42000"},
	Isc_dsql_derived_alias_select: {"Internal dsql error: alias type expected by pass1_expand_select_node\n", "42000"},
	Isc_dsql_derived_alias_field: {"Internal dsql error: alias type expected by pass1_field\n", "42000"},
	Isc_dsql_auto_field_bad_pos: {"Internal dsql error: column position out of range in pass1_union_auto_cast\n", "42000"},
	Isc_dsql_cte_wrong_reference: {"Recursive CTE member (@1) can refer itself only in FROM clause\n", "42000"},
	Isc_dsql_cte_cycle: {"CTE '@1' has cyclic dependencies\n", "42000"},
	Isc_dsql_cte_outer_join: {"Recursive member of CTE can't be member of an outer join\n", "42000"},
	Isc_dsql_cte_mult_references: {"Recursive member of CTE can't reference itself more than once\n", "42000"},
	Isc_dsql_cte_not_a_union: {"Recursive CTE (@1) must be an UNION\n", "42000"},
	Isc_dsql_cte_nonrecurs_after_recurs: {"CTE '@1' defined non-recursive member after recursive\n", "42000"},
	Isc_dsql_cte_wrong_clause: {"Recursive member of CTE '@1' has @2 clause\n", "42000"},
	Isc_dsql_cte_union_all: {"Recursive members of CTE (@1) must be linked with another members via UNION ALL\n", "42000"},
	Isc_dsql_cte_miss_nonrecursive: {"Non-recursive member is missing in CTE '@1'\n", "42000"},
	Isc_dsql_cte_nested_with: {"WITH clause can't be nested\n", "42000"},
	Isc_dsql_col_more_than_once_using: {"column @1 appears more than once in USING clause\n", "42000"},
	Isc_dsql_unsupp_feature_dialect: {"feature is not supported in dialect @1\n", "HY000"},
	Isc_dsql_cte_not_used: {"CTE \"@1\" is not used in query\n", "42000"},
	Isc_dsql_col_more_than_once_view: {"column @1 appears more than once in ALTER VIEW\n", "42000"},
	Isc_dsql_unsupported_in_auto_trans: {"@1 is not supported inside IN AUTONOMOUS TRANSACTION block\n", "HY000"},
	Isc_dsql_eval_unknode: {"Unknown node type @1 in dsql/GEN_expr\n", "42000"},
	Isc_dsql_agg_wrongarg: {"Argument for @1 in dialect 1 must be string or numeric\n", "42000"},
	Isc_dsql_agg2_wrongarg: {"Argument for @1 in dialect 3 must be numeric\n", "42000"},
	Isc_dsql_nodateortime_pm_string: {"Strings cannot be added to or subtracted from DATE or TIME types\n", "42000"},
	Isc_dsql_invalid_datetime_subtract: {"Invalid data type for subtraction involving DATE, TIME or TIMESTAMP types\n", "42000"},
	Isc_dsql_invalid_dateortime_add: {"Adding two DATE values or two TIME values is not allowed\n", "42000"},
	Isc_dsql_invalid_type_minus_date: {"DATE value cannot be subtracted from the provided data type\n", "42000"},
	Isc_dsql_nostring_addsub_dial3: {"Strings cannot be added or subtracted in dialect 3\n", "42000"},
	Isc_dsql_invalid_type_addsub_dial3: {"Invalid data type for addition or subtraction in dialect 3\n", "42000"},
	Isc_dsql_invalid_type_multip_dial1: {"Invalid data type for multiplication in dialect 1\n", "42000"},
	Isc_dsql_nostring_multip_dial3: {"Strings cannot be multiplied in dialect 3\n", "42000"},
	Isc_dsql_invalid_type_multip_dial3: {"Invalid data type for multiplication in dialect 3\n", "42000"},
	Isc_dsql_mustuse_numeric_div_dial1: {"Division in dialect 1 must be between numeric data types\n", "42000"},
	Isc_dsql_nostring_div_dial3: {"Strings cannot be divided in dialect 3\n", "42000"},
	Isc_dsql_invalid_type_div_dial3: {"Invalid data type for division in dialect 3\n", "42000"},
	Isc_dsql_nostring_neg_dial3: {"Strings cannot be negated (applied the minus operator) in dialect 3\n", "42000"},
	Isc_dsql_invalid_type_neg: {"Invalid data type for negation (minus operator)\n", "42000"},
	Isc_dsql_max_distinct_items: {"Cannot have more than 255 items in DISTINCT / UNION DISTINCT list\n", "54011"},
	Isc_dsql_alter_charset_failed: {"ALTER CHARACTER SET @1 failed\n", "42000"},
	Isc_dsql_comment_on_failed: {"COMMENT ON @1 failed\n", "42000"},
	Isc_dsql_create_func_failed: {"CREATE FUNCTION @1 failed\n", "42000"},
	Isc_dsql_alter_func_failed: {"ALTER FUNCTION @1 failed\n", "42000"},
	Isc_dsql_create_alter_func_failed: {"CREATE OR ALTER FUNCTION @1 failed\n", "42000"},
	Isc_dsql_drop_func_failed: {"DROP FUNCTION @1 failed\n", "42000"},
	Isc_dsql_recreate_func_failed: {"RECREATE FUNCTION @1 failed\n", "42000"},
	Isc_dsql_create_proc_failed: {"CREATE PROCEDURE @1 failed\n", "42000"},
	Isc_dsql_alter_proc_failed: {"ALTER PROCEDURE @1 failed\n", "42000"},
	Isc_dsql_create_alter_proc_failed: {"CREATE OR ALTER PROCEDURE @1 failed\n", "42000"},
	Isc_dsql_drop_proc_failed: {"DROP PROCEDURE @1 failed\n", "42000"},
	Isc_dsql_recreate_proc_failed: {"RECREATE PROCEDURE @1 failed\n", "42000"},
	Isc_dsql_create_trigger_failed: {"CREATE TRIGGER @1 failed\n", "42000"},
	Isc_dsql_alter_trigger_failed: {"ALTER TRIGGER @1 failed\n", "42000"},
	Isc_dsql_create_alter_trigger_failed: {"CREATE OR ALTER TRIGGER @1 failed\n", "42000"},
	Isc_dsql_drop_trigger_failed: {"DROP TRIGGER @1 failed\n", "42000"},
	Isc_dsql_recreate_trigger_failed: {"RECREATE TRIGGER @1 failed\n", "42000"},
	Isc_dsql_create_collation_failed: {"CREATE COLLATION @1 failed\n", "42000"},
	Isc_dsql_drop_collation_failed: {"DROP COLLATION @1 failed\n", "42000"},
	Isc_dsql_create_domain_failed: {"CREATE DOMAIN @1 failed\n", "42000"},
	Isc_dsql_alter_domain_failed: {"ALTER DOMAIN @1 failed\n", "42000"},
	Isc_dsql_drop_domain_failed: {"DROP DOMAIN @1 failed\n", "42000"},
	Isc_dsql_create_except_failed: {"CREATE EXCEPTION @1 failed\n", "42000"},
	Isc_dsql_alter_except_failed: {"ALTER EXCEPTION @1 failed\n", "42000"},
	Isc_dsql_create_alter_except_failed: {"CREATE OR ALTER EXCEPTION @1 failed\n", "42000"},
	Isc_dsql_recreate_except_failed: {"RECREATE EXCEPTION @1 failed\n", "42000"},
	Isc_dsql_drop_except_failed: {"DROP EXCEPTION @1 failed\n", "42000"},
	Isc_dsql_create_sequence_failed: {"CREATE SEQUENCE @1 failed\n", "42000"},
	Isc_dsql_create_table_failed: {"CREATE TABLE @1 failed\n", "42000"},
	Isc_dsql_alter_table_failed: {"ALTER TABLE @1 failed\n", "42000"},
	Isc_dsql_drop_table_failed: {"DROP TABLE @1 failed\n", "42000"},
	Isc_dsql_recreate_table_failed: {"RECREATE TABLE @1 failed\n", "42000"},
	Isc_dsql_create_pack_failed: {"CREATE PACKAGE @1 failed\n", "42000"},
	Isc_dsql_alter_pack_failed: {"ALTER PACKAGE @1 failed\n", "42000"},
	Isc_dsql_create_alter_pack_failed: {"CREATE OR ALTER PACKAGE @1 failed\n", "42000"},
	Isc_dsql_drop_pack_failed: {"DROP PACKAGE @1 failed\n", "42000"},
	Isc_dsql_recreate_pack_failed: {"RECREATE PACKAGE @1 failed\n", "42000"},
	Isc_dsql_create_pack_body_failed: {"CREATE PACKAGE BODY @1 failed\n", "42000"},
	Isc_dsql_drop_pack_body_failed: {"DROP PACKAGE BODY @1 failed\n", "42000"},
	Isc_dsql_recreate_pack_body_failed: {"RECREATE PACKAGE BODY @1 failed\n", "42000"},
	Isc_dsql_create_view_failed: {"CREATE VIEW @1 failed\n", "42000"},
	Isc_dsql_alter_view_failed: {"ALTER VIEW @1 failed\n", "42000"},
	Isc_dsql_create_alter_view_failed: {"CREATE OR ALTER VIEW @1 failed\n", "42000"},
	Isc_dsql_recreate_view_failed: {"RECREATE VIEW @1 failed\n", "42000"},
	Isc_dsql_drop_view_failed: {"DROP VIEW @1 failed\n", "42000"},
	Isc_dsql_drop_sequence_failed: {"DROP SEQUENCE @1 failed\n", "42000"},
	Isc_dsql_recreate_sequence_failed: {"RECREATE SEQUENCE @1 failed\n", "42000"},
	Isc_dsql_drop_index_failed: {"DROP INDEX @1 failed\n", "42000"},
	Isc_dsql_drop_filter_failed: {"DROP FILTER @1 failed\n", "42000"},
	Isc_dsql_drop_shadow_failed: {"DROP SHADOW @1 failed\n", "42000"},
	Isc_dsql_drop_role_failed: {"DROP ROLE @1 failed\n", "42000"},
	Isc_dsql_drop_user_failed: {"DROP USER @1 failed\n", "42000"},
	Isc_dsql_create_role_failed: {"CREATE ROLE @1 failed\n", "42000"},
	Isc_dsql_alter_role_failed: {"ALTER ROLE @1 failed\n", "42000"},
	Isc_dsql_alter_index_failed: {"ALTER INDEX @1 failed\n", "42000"},
	Isc_dsql_alter_database_failed: {"ALTER DATABASE failed\n", "42000"},
	Isc_dsql_create_shadow_failed: {"CREATE SHADOW @1 failed\n", "42000"},
	Isc_dsql_create_filter_failed: {"DECLARE FILTER @1 failed\n", "42000"},
	Isc_dsql_create_index_failed: {"CREATE INDEX @1 failed\n", "42000"},
	Isc_dsql_create_user_failed: {"CREATE USER @1 failed\n", "42000"},
	Isc_dsql_alter_user_failed: {"ALTER USER @1 failed\n", "42000"},
	Isc_dsql_grant_failed: {"GRANT failed\n", "42000"},
	Isc_dsql_revoke_failed: {"REVOKE failed\n", "42000"},
	Isc_dsql_cte_recursive_aggregate: {"Recursive member of CTE cannot use aggregate or window function\n", "42000"},
	Isc_dsql_mapping_failed: {"@2 MAPPING @1 failed\n", "42000"},
	Isc_dsql_alter_sequence_failed: {"ALTER SEQUENCE @1 failed\n", "42000"},
	Isc_dsql_create_generator_failed: {"CREATE GENERATOR @1 failed\n", "42000"},
	Isc_dsql_set_generator_failed: {"SET GENERATOR @1 failed\n", "42000"},
	Isc_dsql_wlock_simple: {"WITH LOCK can be used only with a single physical table\n", "42000"},
	Isc_dsql_firstskip_rows: {"FIRST/SKIP cannot be used with OFFSET/FETCH or ROWS\n", "42000"},
	Isc_dsql_wlock_aggregates: {"WITH LOCK cannot be used with aggregates\n", "42000"},
	Isc_dsql_wlock_conflict: {"WITH LOCK cannot be used with @1\n", "42000"},
	Isc_dsql_max_exception_arguments: {"Number of arguments (@1) exceeds the maximum (@2) number of EXCEPTION USING arguments\n", "07002"},
	Isc_dsql_string_byte_length: {"String literal with @1 bytes exceeds the maximum length of @2 bytes\n", "42000"},
	Isc_dsql_string_char_length: {"String literal with @1 characters exceeds the maximum length of @2 characters for the @3 character set\n", "42000"},
	Isc_dsql_max_nesting: {"Too many BEGIN...END nesting. Maximum level is @1\n", "07002"},
	Isc_dsql_recreate_user_failed: {"RECREATE USER @1 failed\n", "42000"},
	Isc_dsql_table_value_many_columns: {"the number of fields exceeds the limit for the @1 operator. Expected @2, received @3\n", "54001"},
	Isc_dsql_create_schema_failed: {"CREATE SCHEMA @1 failed\n", "42000"},
	Isc_dsql_drop_schema_failed: {"DROP SCHEMA @1 failed\n", "42000"},
	Isc_dsql_recreate_schema_failed: {"RECREATE SCHEMA @1 failed\n", "42000"},
	Isc_dsql_alter_schema_failed: {"ALTER SCHEMA @1 failed\n", "42000"},
	Isc_dsql_create_alter_schema_failed: {"CREATE OR ALTER SCHEMA @1 failed\n", "42000"},
	Isc_dsql_create_policy_failed: {"CREATE POLICY @1 failed\n", "42000"},
	Isc_dsql_alter_policy_failed: {"ALTER POLICY @1 failed\n", "42000"},
	Isc_dsql_drop_policy_failed: {"DROP POLICY @1 failed\n", "42000"},
	Isc_dsql_agg_wrongtype: {"Argument for @1 must be field node\n", "42000"},
	Isc_dsql_create_job_failed: {"CREATE JOB @1 failed\n", "42000"},
	Isc_dsql_alter_job_failed: {"ALTER JOB @1 failed\n", "42000"},
	Isc_dsql_create_alter_job_failed: {"CREATE OR ALTER JOB @1 failed\n", "42000"},
	Isc_dsql_drop_job_failed: {"DROP JOB @1 failed\n", "42000"},
	Isc_dsql_recreate_job_failed: {"RECREATE JOB @1 failed\n", "42000"},
	Isc_dsql_create_ts_failed: {"CREATE TABLESPACE @1 failed\n", "42000"},
	Isc_dsql_alter_ts_failed: {"ALTER TABLESPACE @1 failed\n", "42000"},
	Isc_dsql_create_alter_ts_failed: {"CREATE OR ALTER TABLESPACE @1 failed\n", "42000"},
	Isc_dsql_drop_ts_failed: {"DROP TABLESPACE @1 failed\n", "42000"},
	Isc_dsql_recreate_ts_failed: {"RECREATE TABLESPACE @1 failed\n", "42000"},
	Isc_dsql_create_foreign_server_failed: {"CREATE SERVER @1 failed\n", "42000"},
	Isc_dsql_alter_foreign_server_failed: {"ALTER SERVER @1 failed\n", "42000"},
	Isc_dsql_create_alter_foreign_server_failed: {"CREATE OR ALTER SERVER @1 failed\n", "42000"},
	Isc_dsql_recreate_foreign_server_failed: {"RECREATE SERVER @1 failed\n", "42000"},
	Isc_dsql_drop_foreign_server_failed: {"DROP SERVER @1 failed\n", "42000"},
	Isc_dsql_create_user_mapping_failed: {"CREATE USER MAPPING FOR USER @1 AND SERVER @2 failed\n", "42000"},
	Isc_dsql_alter_user_mapping_failed: {"ALTER USER MAPPING FOR USER @1 AND SERVER @2 failed\n", "42000"},
	Isc_dsql_create_alter_user_mapping_failed: {"CREATE OR ALTER USER MAPPING FOR USER @1 AND SERVER @2 failed\n", "42000"},
	Isc_dsql_recreate_user_mapping_failed: {"RECREATE USER MAPPING FOR USER @1 AND SERVER @2 failed\n", "42000"},
	Isc_dsql_drop_user_mapping_failed: {"DROP USER MAPPING FOR USER @1 AND SERVER @2 failed\n", "42000"},
	Isc_dsql_foreign_table_not_found: {"Foreign table @1 does not exist\n", "42S02"},
	Isc_dsql_create_foreign_table_failed: {"CREATE FOREIGN TABLE @1 failed\n", "42000"},
	Isc_dsql_alter_foreign_table_failed: {"ALTER FOREIGN TABLE @1 failed\n", "42000"},
	Isc_dsql_drop_foreign_table_failed: {"DROP FOREIGN TABLE @1 failed\n", "42000"},
	Isc_dsql_recreate_foreign_table_failed: {"RECREATE FOREIGN TABLE @1 failed\n", "42000"},
	Isc_gsec_cant_open_db: {"unable to open database\n", "00000"},
	Isc_gsec_switches_error: {"error in switch specifications\n", "00000"},
	Isc_gsec_no_op_spec: {"no operation specified\n", "00000"},
	Isc_gsec_no_usr_name: {"no user name specified\n", "00000"},
	Isc_gsec_err_add: {"add record error\n", "00000"},
	Isc_gsec_err_modify: {"modify record error\n", "00000"},
	Isc_gsec_err_find_mod: {"find/modify record error\n", "00000"},
	Isc_gsec_err_rec_not_found: {"record not found for user: @1\n", "00000"},
	Isc_gsec_err_delete: {"delete record error\n", "00000"},
	Isc_gsec_err_find_del: {"find/delete record error\n", "00000"},
	Isc_gsec_err_find_disp: {"find/display record error\n", "00000"},
	Isc_gsec_inv_param: {"invalid parameter, no switch defined\n", "00000"},
	Isc_gsec_op_specified: {"operation already specified\n", "00000"},
	Isc_gsec_pw_specified: {"password already specified\n", "00000"},
	Isc_gsec_uid_specified: {"uid already specified\n", "00000"},
	Isc_gsec_gid_specified: {"gid already specified\n", "00000"},
	Isc_gsec_proj_specified: {"project already specified\n", "00000"},
	Isc_gsec_org_specified: {"organization already specified\n", "00000"},
	Isc_gsec_fname_specified: {"first name already specified\n", "00000"},
	Isc_gsec_mname_specified: {"middle name already specified\n", "00000"},
	Isc_gsec_lname_specified: {"last name already specified\n", "00000"},
	Isc_gsec_inv_switch: {"invalid switch specified\n", "00000"},
	Isc_gsec_amb_switch: {"ambiguous switch specified\n", "00000"},
	Isc_gsec_no_op_specified: {"no operation specified for parameters\n", "00000"},
	Isc_gsec_params_not_allowed: {"no parameters allowed for this operation\n", "00000"},
	Isc_gsec_incompat_switch: {"incompatible switches specified\n", "00000"},
	Isc_gsec_inv_username: {"Invalid user name (maximum 31 bytes allowed)\n", "00000"},
	Isc_gsec_inv_pw_length: {"Warning - maximum 8 significant bytes of password used\n", "00000"},
	Isc_gsec_db_specified: {"database already specified\n", "00000"},
	Isc_gsec_db_admin_specified: {"database administrator name already specified\n", "00000"},
	Isc_gsec_db_admin_pw_specified: {"database administrator password already specified\n", "00000"},
	Isc_gsec_sql_role_specified: {"SQL role name already specified\n", "00000"},
	Isc_gstat_unknown_switch: {"found unknown switch\n", "00000"},
	Isc_gstat_retry: {"please retry, giving a database name\n", "00000"},
	Isc_gstat_wrong_ods: {"Wrong ODS version, expected @1, encountered @2\n", "00000"},
	Isc_gstat_unexpected_eof: {"Unexpected end of database file.\n", "00000"},
	Isc_gstat_open_err: {"Can't open database file @1\n", "00000"},
	Isc_gstat_read_err: {"Can't read a database page\n", "00000"},
	Isc_gstat_sysmemex: {"System memory exhausted\n", "00000"},
	Isc_fbsvcmgr_bad_am: {"Wrong value for access mode\n", "00000"},
	Isc_fbsvcmgr_bad_wm: {"Wrong value for write mode\n", "00000"},
	Isc_fbsvcmgr_bad_rs: {"Wrong value for reserve space\n", "00000"},
	Isc_fbsvcmgr_info_err: {"Unknown tag (@1) in info_svr_db_info block after isc_svc_query()\n", "00000"},
	Isc_fbsvcmgr_query_err: {"Unknown tag (@1) in isc_svc_query() results\n", "00000"},
	Isc_fbsvcmgr_switch_unknown: {"Unknown switch \"@1\"\n", "00000"},
	Isc_fbsvcmgr_bad_sm: {"Wrong value for shutdown mode\n", "00000"},
	Isc_fbsvcmgr_fp_open: {"could not open file @1\n", "00000"},
	Isc_fbsvcmgr_fp_read: {"could not read file @1\n", "00000"},
	Isc_fbsvcmgr_fp_empty: {"empty file @1\n", "00000"},
	Isc_fbsvcmgr_bad_arg: {"Invalid or missing parameter for switch @1\n", "00000"},
	Isc_fbsvcmgr_info_limbo: {"Unknown tag (@1) in isc_info_svc_limbo_trans block after isc_svc_query()\n", "00000"},
	Isc_fbsvcmgr_limbo_state: {"Unknown tag (@1) in isc_spb_tra_state block after isc_svc_query()\n", "00000"},
	Isc_fbsvcmgr_limbo_advise: {"Unknown tag (@1) in isc_spb_tra_advise block after isc_svc_query()\n", "00000"},
	Isc_fbsvcmgr_bad_rm: {"Wrong value for replica mode\n", "00000"},
	Isc_utl_trusted_switch: {"Switches trusted_user and trusted_role are not supported from command line\n", "00000"},
	Isc_nbackup_missing_param: {"Missing parameter for switch @1\n", "00000"},
	Isc_nbackup_allowed_switches: {"Only one of -LOCK, -UNLOCK, -FIXUP, -BACKUP or -RESTORE should be specified\n", "00000"},
	Isc_nbackup_unknown_param: {"Unrecognized parameter @1\n", "00000"},
	Isc_nbackup_unknown_switch: {"Unknown switch @1\n", "00000"},
	Isc_nbackup_nofetchpw_svc: {"Fetch password can't be used in service mode\n", "00000"},
	Isc_nbackup_pwfile_error: {"Error working with password file \"@1\"\n", "00000"},
	Isc_nbackup_size_with_lock: {"Switch -SIZE can be used only with -LOCK\n", "00000"},
	Isc_nbackup_no_switch: {"None of -LOCK, -UNLOCK, -FIXUP, -BACKUP or -RESTORE specified\n", "00000"},
	Isc_nbackup_err_read: {"IO error reading file: @1\n", "00000"},
	Isc_nbackup_err_write: {"IO error writing file: @1\n", "00000"},
	Isc_nbackup_err_seek: {"IO error seeking file: @1\n", "00000"},
	Isc_nbackup_err_opendb: {"Error opening database file: @1\n", "00000"},
	Isc_nbackup_err_fadvice: {"Error in posix_fadvise(@1) for database @2\n", "00000"},
	Isc_nbackup_err_createdb: {"Error creating database file: @1\n", "00000"},
	Isc_nbackup_err_openbk: {"Error opening backup file: @1\n", "00000"},
	Isc_nbackup_err_createbk: {"Error creating backup file: @1\n", "00000"},
	Isc_nbackup_err_eofdb: {"Unexpected end of database file @1\n", "00000"},
	Isc_nbackup_fixup_wrongstate: {"Database @1 is not in state (@2) to be safely fixed up\n", "00000"},
	Isc_nbackup_err_db: {"Database error\n", "00000"},
	Isc_nbackup_userpw_toolong: {"Username or password is too long\n", "00000"},
	Isc_nbackup_lostrec_db: {"Cannot find record for database \"@1\" backup level @2 in the backup history\n", "00000"},
	Isc_nbackup_lostguid_db: {"Internal error. History query returned null SCN or GUID\n", "00000"},
	Isc_nbackup_err_eofhdrdb: {"Unexpected end of file when reading header of database file \"@1\" (stage @2)\n", "00000"},
	Isc_nbackup_db_notlock: {"Internal error. Database file is not locked. Flags are @1\n", "00000"},
	Isc_nbackup_lostguid_bk: {"Internal error. Cannot get backup guid clumplet\n", "00000"},
	Isc_nbackup_page_changed: {"Internal error. Database page @1 had been changed during backup (page SCN=@2, backup SCN=@3)\n", "00000"},
	Isc_nbackup_dbsize_inconsistent: {"Database file size is not a multiple of page size\n", "00000"},
	Isc_nbackup_failed_lzbk: {"Level 0 backup is not restored\n", "00000"},
	Isc_nbackup_err_eofhdrbk: {"Unexpected end of file when reading header of backup file: @1\n", "00000"},
	Isc_nbackup_invalid_incbk: {"Invalid incremental backup file: @1\n", "00000"},
	Isc_nbackup_unsupvers_incbk: {"Unsupported version @1 of incremental backup file: @2\n", "00000"},
	Isc_nbackup_invlevel_incbk: {"Invalid level @1 of incremental backup file: @2, expected @3\n", "00000"},
	Isc_nbackup_wrong_orderbk: {"Wrong order of backup files or invalid incremental backup file detected, file: @1\n", "00000"},
	Isc_nbackup_err_eofbk: {"Unexpected end of backup file: @1\n", "00000"},
	Isc_nbackup_err_copy: {"Error creating database file: @1 via copying from: @2\n", "00000"},
	Isc_nbackup_err_eofhdr_restdb: {"Unexpected end of file when reading header of restored database file (stage @1)\n", "00000"},
	Isc_nbackup_lostguid_l0bk: {"Cannot get backup guid clumplet from L0 backup\n", "00000"},
	Isc_nbackup_switchd_parameter: {"Wrong parameter @1 for switch -D, need ON or OFF\n", "00000"},
	Isc_nbackup_user_stop: {"Terminated due to user request\n", "08006"},
	Isc_nbackup_deco_parse: {"Too complex decompress command (> @1 arguments)\n", "54023"},
	Isc_nbackup_lostrec_guid_db: {"Cannot find record for database \"@1\" backup GUID @2 in the backup history\n", "00000"},
	Isc_nbackup_seq_misuse: {"Switch -SEQ(UENCE) can be used only with -FIXUP or -RESTORE\n", "00000"},
	Isc_nbackup_wrong_param: {"Wrong parameter value for switch @1\n", "00000"},
	Isc_nbackup_clean_hist_misuse: {"Switch -CLEAN_HISTORY can be used only with -BACKUP\n", "00000"},
	Isc_nbackup_clean_hist_missed: {"-KEEP can be used only with -CLEAN_HISTORY\n", "00000"},
	Isc_nbackup_keep_hist_missed: {"-KEEP is required with -CLEAN_HISTORY\n", "00000"},
	Isc_nbackup_second_keep_switch: {"-KEEP can be used one time only\n", "00000"},
	Isc_trace_conflict_acts: {"conflicting actions \"@1\" and \"@2\" found\n", "00000"},
	Isc_trace_act_notfound: {"action switch not found\n", "00000"},
	Isc_trace_switch_once: {"switch \"@1\" must be set only once\n", "00000"},
	Isc_trace_param_val_miss: {"value for switch \"@1\" is missing\n", "00000"},
	Isc_trace_param_invalid: {"invalid value (\"@1\") for switch \"@2\"\n", "00000"},
	Isc_trace_switch_unknown: {"unknown switch \"@1\" encountered\n", "00000"},
	Isc_trace_switch_svc_only: {"switch \"@1\" can be used by service only\n", "00000"},
	Isc_trace_switch_user_only: {"switch \"@1\" can be used by interactive user only\n", "00000"},
	Isc_trace_switch_param_miss: {"mandatory parameter \"@1\" for switch \"@2\" is missing\n", "00000"},
	Isc_trace_param_act_notcompat: {"parameter \"@1\" is incompatible with action \"@2\"\n", "00000"},
	Isc_trace_mandatory_switch_miss: {"mandatory switch \"@1\" is missing\n", "00000"},
	Isc_diff_incmp_sw: {"incompatible switch combination\n", "00000"},
	Isc_diff_unicode_init_error: {"Cannot initialize UNICODE collation\n", "00000"},
	Isc_diff_replconf_section_not_found: {"Replication config: @1 section not found\n", "00000"},
	Isc_diff_param_val_miss: {"Value for switch \"@1\" is missing\n", "00000"},
}
