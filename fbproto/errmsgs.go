// Code generated from _tools/msgs DO NOT EDIT.
package fbproto

type fberror struct {
	message  string
	params   int
	sqlstate string
}

const (
	Isc_arith_except                                              = 335544321
	Isc_bad_dbkey                                                 = 335544322
	Isc_bad_db_format                                             = 335544323
	Isc_bad_db_handle                                             = 335544324
	Isc_bad_dpb_content                                           = 335544325
	Isc_bad_dpb_form                                              = 335544326
	Isc_bad_req_handle                                            = 335544327
	Isc_bad_segstr_handle                                         = 335544328
	Isc_bad_segstr_id                                             = 335544329
	Isc_bad_tpb_content                                           = 335544330
	Isc_bad_tpb_form                                              = 335544331
	Isc_bad_trans_handle                                          = 335544332
	Isc_bug_check                                                 = 335544333
	Isc_convert_error                                             = 335544334
	Isc_db_corrupt                                                = 335544335
	Isc_deadlock                                                  = 335544336
	Isc_excess_trans                                              = 335544337
	Isc_from_no_match                                             = 335544338
	Isc_infinap                                                   = 335544339
	Isc_infona                                                    = 335544340
	Isc_infunk                                                    = 335544341
	Isc_integ_fail                                                = 335544342
	Isc_invalid_blr                                               = 335544343
	Isc_io_error                                                  = 335544344
	Isc_lock_conflict                                             = 335544345
	Isc_metadata_corrupt                                          = 335544346
	Isc_not_valid                                                 = 335544347
	Isc_no_cur_rec                                                = 335544348
	Isc_no_dup                                                    = 335544349
	Isc_no_finish                                                 = 335544350
	Isc_no_meta_update                                            = 335544351
	Isc_no_priv                                                   = 335544352
	Isc_no_recon                                                  = 335544353
	Isc_no_record                                                 = 335544354
	Isc_no_segstr_close                                           = 335544355
	Isc_obsolete_metadata                                         = 335544356
	Isc_open_trans                                                = 335544357
	Isc_port_len                                                  = 335544358
	Isc_read_only_field                                           = 335544359
	Isc_read_only_rel                                             = 335544360
	Isc_read_only_trans                                           = 335544361
	Isc_read_only_view                                            = 335544362
	Isc_req_no_trans                                              = 335544363
	Isc_req_sync                                                  = 335544364
	Isc_req_wrong_db                                              = 335544365
	Isc_segment                                                   = 335544366
	Isc_segstr_eof                                                = 335544367
	Isc_segstr_no_op                                              = 335544368
	Isc_segstr_no_read                                            = 335544369
	Isc_segstr_no_trans                                           = 335544370
	Isc_segstr_no_write                                           = 335544371
	Isc_segstr_wrong_db                                           = 335544372
	Isc_sys_request                                               = 335544373
	Isc_stream_eof                                                = 335544374
	Isc_unavailable                                               = 335544375
	Isc_unres_rel                                                 = 335544376
	Isc_uns_ext                                                   = 335544377
	Isc_wish_list                                                 = 335544378
	Isc_wrong_ods                                                 = 335544379
	Isc_wronumarg                                                 = 335544380
	Isc_imp_exc                                                   = 335544381
	Isc_random                                                    = 335544382
	Isc_fatal_conflict                                            = 335544383
	Isc_badblk                                                    = 335544384
	Isc_invpoolcl                                                 = 335544385
	Isc_nopoolids                                                 = 335544386
	Isc_relbadblk                                                 = 335544387
	Isc_blktoobig                                                 = 335544388
	Isc_bufexh                                                    = 335544389
	Isc_syntaxerr                                                 = 335544390
	Isc_bufinuse                                                  = 335544391
	Isc_bdbincon                                                  = 335544392
	Isc_reqinuse                                                  = 335544393
	Isc_badodsver                                                 = 335544394
	Isc_relnotdef                                                 = 335544395
	Isc_fldnotdef                                                 = 335544396
	Isc_dirtypage                                                 = 335544397
	Isc_waifortra                                                 = 335544398
	Isc_doubleloc                                                 = 335544399
	Isc_nodnotfnd                                                 = 335544400
	Isc_dupnodfnd                                                 = 335544401
	Isc_locnotmar                                                 = 335544402
	Isc_badpagtyp                                                 = 335544403
	Isc_corrupt                                                   = 335544404
	Isc_badpage                                                   = 335544405
	Isc_badindex                                                  = 335544406
	Isc_dbbnotzer                                                 = 335544407
	Isc_tranotzer                                                 = 335544408
	Isc_trareqmis                                                 = 335544409
	Isc_badhndcnt                                                 = 335544410
	Isc_wrotpbver                                                 = 335544411
	Isc_wroblrver                                                 = 335544412
	Isc_wrodpbver                                                 = 335544413
	Isc_blobnotsup                                                = 335544414
	Isc_badrelation                                               = 335544415
	Isc_nodetach                                                  = 335544416
	Isc_notremote                                                 = 335544417
	Isc_trainlim                                                  = 335544418
	Isc_notinlim                                                  = 335544419
	Isc_traoutsta                                                 = 335544420
	Isc_connect_reject                                            = 335544421
	Isc_dbfile                                                    = 335544422
	Isc_orphan                                                    = 335544423
	Isc_no_lock_mgr                                               = 335544424
	Isc_ctxinuse                                                  = 335544425
	Isc_ctxnotdef                                                 = 335544426
	Isc_datnotsup                                                 = 335544427
	Isc_badmsgnum                                                 = 335544428
	Isc_badparnum                                                 = 335544429
	Isc_virmemexh                                                 = 335544430
	Isc_blocking_signal                                           = 335544431
	Isc_lockmanerr                                                = 335544432
	Isc_journerr                                                  = 335544433
	Isc_keytoobig                                                 = 335544434
	Isc_nullsegkey                                                = 335544435
	Isc_sqlerr                                                    = 335544436
	Isc_wrodynver                                                 = 335544437
	Isc_funnotdef                                                 = 335544438
	Isc_funmismat                                                 = 335544439
	Isc_bad_msg_vec                                               = 335544440
	Isc_bad_detach                                                = 335544441
	Isc_noargacc_read                                             = 335544442
	Isc_noargacc_write                                            = 335544443
	Isc_read_only                                                 = 335544444
	Isc_ext_err                                                   = 335544445
	Isc_non_updatable                                             = 335544446
	Isc_no_rollback                                               = 335544447
	Isc_bad_sec_info                                              = 335544448
	Isc_invalid_sec_info                                          = 335544449
	Isc_misc_interpreted                                          = 335544450
	Isc_update_conflict                                           = 335544451
	Isc_unlicensed                                                = 335544452
	Isc_obj_in_use                                                = 335544453
	Isc_nofilter                                                  = 335544454
	Isc_shadow_accessed                                           = 335544455
	Isc_invalid_sdl                                               = 335544456
	Isc_out_of_bounds                                             = 335544457
	Isc_invalid_dimension                                         = 335544458
	Isc_rec_in_limbo                                              = 335544459
	Isc_shadow_missing                                            = 335544460
	Isc_cant_validate                                             = 335544461
	Isc_cant_start_journal                                        = 335544462
	Isc_gennotdef                                                 = 335544463
	Isc_cant_start_logging                                        = 335544464
	Isc_bad_segstr_type                                           = 335544465
	Isc_foreign_key                                               = 335544466
	Isc_high_minor                                                = 335544467
	Isc_tra_state                                                 = 335544468
	Isc_trans_invalid                                             = 335544469
	Isc_buf_invalid                                               = 335544470
	Isc_indexnotdefined                                           = 335544471
	Isc_login                                                     = 335544472
	Isc_invalid_bookmark                                          = 335544473
	Isc_bad_lock_level                                            = 335544474
	Isc_relation_lock                                             = 335544475
	Isc_record_lock                                               = 335544476
	Isc_max_idx                                                   = 335544477
	Isc_jrn_enable                                                = 335544478
	Isc_old_failure                                               = 335544479
	Isc_old_in_progress                                           = 335544480
	Isc_old_no_space                                              = 335544481
	Isc_no_wal_no_jrn                                             = 335544482
	Isc_num_old_files                                             = 335544483
	Isc_wal_file_open                                             = 335544484
	Isc_bad_stmt_handle                                           = 335544485
	Isc_wal_failure                                               = 335544486
	Isc_walw_err                                                  = 335544487
	Isc_logh_small                                                = 335544488
	Isc_logh_inv_version                                          = 335544489
	Isc_logh_open_flag                                            = 335544490
	Isc_logh_open_flag2                                           = 335544491
	Isc_logh_diff_dbname                                          = 335544492
	Isc_logf_unexpected_eof                                       = 335544493
	Isc_logr_incomplete                                           = 335544494
	Isc_logr_header_small                                         = 335544495
	Isc_logb_small                                                = 335544496
	Isc_wal_illegal_attach                                        = 335544497
	Isc_wal_invalid_wpb                                           = 335544498
	Isc_wal_err_rollover                                          = 335544499
	Isc_no_wal                                                    = 335544500
	Isc_drop_wal                                                  = 335544501
	Isc_stream_not_defined                                        = 335544502
	Isc_wal_subsys_error                                          = 335544503
	Isc_wal_subsys_corrupt                                        = 335544504
	Isc_no_archive                                                = 335544505
	Isc_shutinprog                                                = 335544506
	Isc_range_in_use                                              = 335544507
	Isc_range_not_found                                           = 335544508
	Isc_charset_not_found                                         = 335544509
	Isc_lock_timeout                                              = 335544510
	Isc_prcnotdef                                                 = 335544511
	Isc_prcmismat                                                 = 335544512
	Isc_wal_bugcheck                                              = 335544513
	Isc_wal_cant_expand                                           = 335544514
	Isc_codnotdef                                                 = 335544515
	Isc_xcpnotdef                                                 = 335544516
	Isc_except                                                    = 335544517
	Isc_cache_restart                                             = 335544518
	Isc_bad_lock_handle                                           = 335544519
	Isc_jrn_present                                               = 335544520
	Isc_wal_err_rollover2                                         = 335544521
	Isc_wal_err_logwrite                                          = 335544522
	Isc_wal_err_jrn_comm                                          = 335544523
	Isc_wal_err_expansion                                         = 335544524
	Isc_wal_err_setup                                             = 335544525
	Isc_wal_err_ww_sync                                           = 335544526
	Isc_wal_err_ww_start                                          = 335544527
	Isc_shutdown                                                  = 335544528
	Isc_existing_priv_mod                                         = 335544529
	Isc_primary_key_ref                                           = 335544530
	Isc_primary_key_notnull                                       = 335544531
	Isc_ref_cnstrnt_notfound                                      = 335544532
	Isc_foreign_key_notfound                                      = 335544533
	Isc_ref_cnstrnt_update                                        = 335544534
	Isc_check_cnstrnt_update                                      = 335544535
	Isc_check_cnstrnt_del                                         = 335544536
	Isc_integ_index_seg_del                                       = 335544537
	Isc_integ_index_seg_mod                                       = 335544538
	Isc_integ_index_del                                           = 335544539
	Isc_integ_index_mod                                           = 335544540
	Isc_check_trig_del                                            = 335544541
	Isc_check_trig_update                                         = 335544542
	Isc_cnstrnt_fld_del                                           = 335544543
	Isc_cnstrnt_fld_rename                                        = 335544544
	Isc_rel_cnstrnt_update                                        = 335544545
	Isc_constaint_on_view                                         = 335544546
	Isc_invld_cnstrnt_type                                        = 335544547
	Isc_primary_key_exists                                        = 335544548
	Isc_systrig_update                                            = 335544549
	Isc_not_rel_owner                                             = 335544550
	Isc_grant_obj_notfound                                        = 335544551
	Isc_grant_fld_notfound                                        = 335544552
	Isc_grant_nopriv                                              = 335544553
	Isc_nonsql_security_rel                                       = 335544554
	Isc_nonsql_security_fld                                       = 335544555
	Isc_wal_cache_err                                             = 335544556
	Isc_shutfail                                                  = 335544557
	Isc_check_constraint                                          = 335544558
	Isc_bad_svc_handle                                            = 335544559
	Isc_shutwarn                                                  = 335544560
	Isc_wrospbver                                                 = 335544561
	Isc_bad_spb_form                                              = 335544562
	Isc_svcnotdef                                                 = 335544563
	Isc_no_jrn                                                    = 335544564
	Isc_transliteration_failed                                    = 335544565
	Isc_start_cm_for_wal                                          = 335544566
	Isc_wal_ovflow_log_required                                   = 335544567
	Isc_text_subtype                                              = 335544568
	Isc_dsql_error                                                = 335544569
	Isc_dsql_command_err                                          = 335544570
	Isc_dsql_constant_err                                         = 335544571
	Isc_dsql_cursor_err                                           = 335544572
	Isc_dsql_datatype_err                                         = 335544573
	Isc_dsql_decl_err                                             = 335544574
	Isc_dsql_cursor_update_err                                    = 335544575
	Isc_dsql_cursor_open_err                                      = 335544576
	Isc_dsql_cursor_close_err                                     = 335544577
	Isc_dsql_field_err                                            = 335544578
	Isc_dsql_internal_err                                         = 335544579
	Isc_dsql_relation_err                                         = 335544580
	Isc_dsql_procedure_err                                        = 335544581
	Isc_dsql_request_err                                          = 335544582
	Isc_dsql_sqlda_err                                            = 335544583
	Isc_dsql_var_count_err                                        = 335544584
	Isc_dsql_stmt_handle                                          = 335544585
	Isc_dsql_function_err                                         = 335544586
	Isc_dsql_blob_err                                             = 335544587
	Isc_collation_not_found                                       = 335544588
	Isc_collation_not_for_charset                                 = 335544589
	Isc_dsql_dup_option                                           = 335544590
	Isc_dsql_tran_err                                             = 335544591
	Isc_dsql_invalid_array                                        = 335544592
	Isc_dsql_max_arr_dim_exceeded                                 = 335544593
	Isc_dsql_arr_range_error                                      = 335544594
	Isc_dsql_trigger_err                                          = 335544595
	Isc_dsql_subselect_err                                        = 335544596
	Isc_dsql_crdb_prepare_err                                     = 335544597
	Isc_specify_field_err                                         = 335544598
	Isc_num_field_err                                             = 335544599
	Isc_col_name_err                                              = 335544600
	Isc_where_err                                                 = 335544601
	Isc_table_view_err                                            = 335544602
	Isc_distinct_err                                              = 335544603
	Isc_key_field_count_err                                       = 335544604
	Isc_subquery_err                                              = 335544605
	Isc_expression_eval_err                                       = 335544606
	Isc_node_err                                                  = 335544607
	Isc_command_end_err                                           = 335544608
	Isc_index_name                                                = 335544609
	Isc_exception_name                                            = 335544610
	Isc_field_name                                                = 335544611
	Isc_token_err                                                 = 335544612
	Isc_union_err                                                 = 335544613
	Isc_dsql_construct_err                                        = 335544614
	Isc_field_aggregate_err                                       = 335544615
	Isc_field_ref_err                                             = 335544616
	Isc_order_by_err                                              = 335544617
	Isc_return_mode_err                                           = 335544618
	Isc_extern_func_err                                           = 335544619
	Isc_alias_conflict_err                                        = 335544620
	Isc_procedure_conflict_error                                  = 335544621
	Isc_relation_conflict_err                                     = 335544622
	Isc_dsql_domain_err                                           = 335544623
	Isc_idx_seg_err                                               = 335544624
	Isc_node_name_err                                             = 335544625
	Isc_table_name                                                = 335544626
	Isc_proc_name                                                 = 335544627
	Isc_idx_create_err                                            = 335544628
	Isc_wal_shadow_err                                            = 335544629
	Isc_dependency                                                = 335544630
	Isc_idx_key_err                                               = 335544631
	Isc_dsql_file_length_err                                      = 335544632
	Isc_dsql_shadow_number_err                                    = 335544633
	Isc_dsql_token_unk_err                                        = 335544634
	Isc_dsql_no_relation_alias                                    = 335544635
	Isc_indexname                                                 = 335544636
	Isc_no_stream_plan                                            = 335544637
	Isc_stream_twice                                              = 335544638
	Isc_stream_not_found                                          = 335544639
	Isc_collation_requires_text                                   = 335544640
	Isc_dsql_domain_not_found                                     = 335544641
	Isc_index_unused                                              = 335544642
	Isc_dsql_self_join                                            = 335544643
	Isc_stream_bof                                                = 335544644
	Isc_stream_crack                                              = 335544645
	Isc_db_or_file_exists                                         = 335544646
	Isc_invalid_operator                                          = 335544647
	Isc_conn_lost                                                 = 335544648
	Isc_bad_checksum                                              = 335544649
	Isc_page_type_err                                             = 335544650
	Isc_ext_readonly_err                                          = 335544651
	Isc_sing_select_err                                           = 335544652
	Isc_psw_attach                                                = 335544653
	Isc_psw_start_trans                                           = 335544654
	Isc_invalid_direction                                         = 335544655
	Isc_dsql_var_conflict                                         = 335544656
	Isc_dsql_no_blob_array                                        = 335544657
	Isc_dsql_base_table                                           = 335544658
	Isc_duplicate_base_table                                      = 335544659
	Isc_view_alias                                                = 335544660
	Isc_index_root_page_full                                      = 335544661
	Isc_dsql_blob_type_unknown                                    = 335544662
	Isc_req_max_clones_exceeded                                   = 335544663
	Isc_dsql_duplicate_spec                                       = 335544664
	Isc_unique_key_violation                                      = 335544665
	Isc_srvr_version_too_old                                      = 335544666
	Isc_drdb_completed_with_errs                                  = 335544667
	Isc_dsql_procedure_use_err                                    = 335544668
	Isc_dsql_count_mismatch                                       = 335544669
	Isc_blob_idx_err                                              = 335544670
	Isc_array_idx_err                                             = 335544671
	Isc_key_field_err                                             = 335544672
	Isc_no_delete                                                 = 335544673
	Isc_del_last_field                                            = 335544674
	Isc_sort_err                                                  = 335544675
	Isc_sort_mem_err                                              = 335544676
	Isc_version_err                                               = 335544677
	Isc_inval_key_posn                                            = 335544678
	Isc_no_segments_err                                           = 335544679
	Isc_crrp_data_err                                             = 335544680
	Isc_rec_size_err                                              = 335544681
	Isc_dsql_field_ref                                            = 335544682
	Isc_req_depth_exceeded                                        = 335544683
	Isc_no_field_access                                           = 335544684
	Isc_no_dbkey                                                  = 335544685
	Isc_jrn_format_err                                            = 335544686
	Isc_jrn_file_full                                             = 335544687
	Isc_dsql_open_cursor_request                                  = 335544688
	Isc_ib_error                                                  = 335544689
	Isc_cache_redef                                               = 335544690
	Isc_cache_too_small                                           = 335544691
	Isc_log_redef                                                 = 335544692
	Isc_log_too_small                                             = 335544693
	Isc_partition_too_small                                       = 335544694
	Isc_partition_not_supp                                        = 335544695
	Isc_log_length_spec                                           = 335544696
	Isc_precision_err                                             = 335544697
	Isc_scale_nogt                                                = 335544698
	Isc_expec_short                                               = 335544699
	Isc_expec_long                                                = 335544700
	Isc_expec_ushort                                              = 335544701
	Isc_escape_invalid                                            = 335544702
	Isc_svcnoexe                                                  = 335544703
	Isc_net_lookup_err                                            = 335544704
	Isc_service_unknown                                           = 335544705
	Isc_host_unknown                                              = 335544706
	Isc_grant_nopriv_on_base                                      = 335544707
	Isc_dyn_fld_ambiguous                                         = 335544708
	Isc_dsql_agg_ref_err                                          = 335544709
	Isc_complex_view                                              = 335544710
	Isc_unprepared_stmt                                           = 335544711
	Isc_expec_positive                                            = 335544712
	Isc_dsql_sqlda_value_err                                      = 335544713
	Isc_invalid_array_id                                          = 335544714
	Isc_extfile_uns_op                                            = 335544715
	Isc_svc_in_use                                                = 335544716
	Isc_err_stack_limit                                           = 335544717
	Isc_invalid_key                                               = 335544718
	Isc_net_init_error                                            = 335544719
	Isc_loadlib_failure                                           = 335544720
	Isc_network_error                                             = 335544721
	Isc_net_connect_err                                           = 335544722
	Isc_net_connect_listen_err                                    = 335544723
	Isc_net_event_connect_err                                     = 335544724
	Isc_net_event_listen_err                                      = 335544725
	Isc_net_read_err                                              = 335544726
	Isc_net_write_err                                             = 335544727
	Isc_integ_index_deactivate                                    = 335544728
	Isc_integ_deactivate_primary                                  = 335544729
	Isc_cse_not_supported                                         = 335544730
	Isc_tra_must_sweep                                            = 335544731
	Isc_unsupported_network_drive                                 = 335544732
	Isc_io_create_err                                             = 335544733
	Isc_io_open_err                                               = 335544734
	Isc_io_close_err                                              = 335544735
	Isc_io_read_err                                               = 335544736
	Isc_io_write_err                                              = 335544737
	Isc_io_delete_err                                             = 335544738
	Isc_io_access_err                                             = 335544739
	Isc_udf_exception                                             = 335544740
	Isc_lost_db_connection                                        = 335544741
	Isc_no_write_user_priv                                        = 335544742
	Isc_token_too_long                                            = 335544743
	Isc_max_att_exceeded                                          = 335544744
	Isc_login_same_as_role_name                                   = 335544745
	Isc_reftable_requires_pk                                      = 335544746
	Isc_usrname_too_long                                          = 335544747
	Isc_password_too_long                                         = 335544748
	Isc_usrname_required                                          = 335544749
	Isc_password_required                                         = 335544750
	Isc_bad_protocol                                              = 335544751
	Isc_dup_usrname_found                                         = 335544752
	Isc_usrname_not_found                                         = 335544753
	Isc_error_adding_sec_record                                   = 335544754
	Isc_error_modifying_sec_record                                = 335544755
	Isc_error_deleting_sec_record                                 = 335544756
	Isc_error_updating_sec_db                                     = 335544757
	Isc_sort_rec_size_err                                         = 335544758
	Isc_bad_default_value                                         = 335544759
	Isc_invalid_clause                                            = 335544760
	Isc_too_many_handles                                          = 335544761
	Isc_optimizer_blk_exc                                         = 335544762
	Isc_invalid_string_constant                                   = 335544763
	Isc_transitional_date                                         = 335544764
	Isc_read_only_database                                        = 335544765
	Isc_must_be_dialect_2_and_up                                  = 335544766
	Isc_blob_filter_exception                                     = 335544767
	Isc_exception_access_violation                                = 335544768
	Isc_exception_datatype_missalignment                          = 335544769
	Isc_exception_array_bounds_exceeded                           = 335544770
	Isc_exception_float_denormal_operand                          = 335544771
	Isc_exception_float_divide_by_zero                            = 335544772
	Isc_exception_float_inexact_result                            = 335544773
	Isc_exception_float_invalid_operand                           = 335544774
	Isc_exception_float_overflow                                  = 335544775
	Isc_exception_float_stack_check                               = 335544776
	Isc_exception_float_underflow                                 = 335544777
	Isc_exception_integer_divide_by_zero                          = 335544778
	Isc_exception_integer_overflow                                = 335544779
	Isc_exception_unknown                                         = 335544780
	Isc_exception_stack_overflow                                  = 335544781
	Isc_exception_sigsegv                                         = 335544782
	Isc_exception_sigill                                          = 335544783
	Isc_exception_sigbus                                          = 335544784
	Isc_exception_sigfpe                                          = 335544785
	Isc_ext_file_delete                                           = 335544786
	Isc_ext_file_modify                                           = 335544787
	Isc_adm_task_denied                                           = 335544788
	Isc_extract_input_mismatch                                    = 335544789
	Isc_insufficient_svc_privileges                               = 335544790
	Isc_file_in_use                                               = 335544791
	Isc_service_att_err                                           = 335544792
	Isc_ddl_not_allowed_by_db_sql_dial                            = 335544793
	Isc_cancelled                                                 = 335544794
	Isc_unexp_spb_form                                            = 335544795
	Isc_sql_dialect_datatype_unsupport                            = 335544796
	Isc_svcnouser                                                 = 335544797
	Isc_depend_on_uncommitted_rel                                 = 335544798
	Isc_svc_name_missing                                          = 335544799
	Isc_too_many_contexts                                         = 335544800
	Isc_datype_notsup                                             = 335544801
	Isc_dialect_reset_warning                                     = 335544802
	Isc_dialect_not_changed                                       = 335544803
	Isc_database_create_failed                                    = 335544804
	Isc_inv_dialect_specified                                     = 335544805
	Isc_valid_db_dialects                                         = 335544806
	Isc_sqlwarn                                                   = 335544807
	Isc_dtype_renamed                                             = 335544808
	Isc_extern_func_dir_error                                     = 335544809
	Isc_date_range_exceeded                                       = 335544810
	Isc_inv_client_dialect_specified                              = 335544811
	Isc_valid_client_dialects                                     = 335544812
	Isc_optimizer_between_err                                     = 335544813
	Isc_service_not_supported                                     = 335544814
	Isc_generator_name                                            = 335544815
	Isc_udf_name                                                  = 335544816
	Isc_bad_limit_param                                           = 335544817
	Isc_bad_skip_param                                            = 335544818
	Isc_io_32bit_exceeded_err                                     = 335544819
	Isc_invalid_savepoint                                         = 335544820
	Isc_dsql_column_pos_err                                       = 335544821
	Isc_dsql_agg_where_err                                        = 335544822
	Isc_dsql_agg_group_err                                        = 335544823
	Isc_dsql_agg_column_err                                       = 335544824
	Isc_dsql_agg_having_err                                       = 335544825
	Isc_dsql_agg_nested_err                                       = 335544826
	Isc_exec_sql_invalid_arg                                      = 335544827
	Isc_exec_sql_invalid_req                                      = 335544828
	Isc_exec_sql_invalid_var                                      = 335544829
	Isc_exec_sql_max_call_exceeded                                = 335544830
	Isc_conf_access_denied                                        = 335544831
	Isc_wrong_backup_state                                        = 335544832
	Isc_wal_backup_err                                            = 335544833
	Isc_cursor_not_open                                           = 335544834
	Isc_bad_shutdown_mode                                         = 335544835
	Isc_concat_overflow                                           = 335544836
	Isc_bad_substring_offset                                      = 335544837
	Isc_foreign_key_target_doesnt_exist                           = 335544838
	Isc_foreign_key_references_present                            = 335544839
	Isc_no_update                                                 = 335544840
	Isc_cursor_already_open                                       = 335544841
	Isc_stack_trace                                               = 335544842
	Isc_ctx_var_not_found                                         = 335544843
	Isc_ctx_namespace_invalid                                     = 335544844
	Isc_ctx_too_big                                               = 335544845
	Isc_ctx_bad_argument                                          = 335544846
	Isc_identifier_too_long                                       = 335544847
	Isc_except2                                                   = 335544848
	Isc_malformed_string                                          = 335544849
	Isc_prc_out_param_mismatch                                    = 335544850
	Isc_command_end_err2                                          = 335544851
	Isc_partner_idx_incompat_type                                 = 335544852
	Isc_bad_substring_length                                      = 335544853
	Isc_charset_not_installed                                     = 335544854
	Isc_collation_not_installed                                   = 335544855
	Isc_att_shutdown                                              = 335544856
	Isc_blobtoobig                                                = 335544857
	Isc_must_have_phys_field                                      = 335544858
	Isc_invalid_time_precision                                    = 335544859
	Isc_blob_convert_error                                        = 335544860
	Isc_array_convert_error                                       = 335544861
	Isc_record_lock_not_supp                                      = 335544862
	Isc_partner_idx_not_found                                     = 335544863
	Isc_tra_num_exc                                               = 335544864
	Isc_field_disappeared                                         = 335544865
	Isc_met_wrong_gtt_scope                                       = 335544866
	Isc_subtype_for_internal_use                                  = 335544867
	Isc_illegal_prc_type                                          = 335544868
	Isc_invalid_sort_datatype                                     = 335544869
	Isc_collation_name                                            = 335544870
	Isc_domain_name                                               = 335544871
	Isc_domnotdef                                                 = 335544872
	Isc_array_max_dimensions                                      = 335544873
	Isc_max_db_per_trans_allowed                                  = 335544874
	Isc_bad_debug_format                                          = 335544875
	Isc_bad_proc_BLR                                              = 335544876
	Isc_key_too_big                                               = 335544877
	Isc_concurrent_transaction                                    = 335544878
	Isc_not_valid_for_var                                         = 335544879
	Isc_not_valid_for                                             = 335544880
	Isc_need_difference                                           = 335544881
	Isc_long_login                                                = 335544882
	Isc_fldnotdef2                                                = 335544883
	Isc_invalid_similar_pattern                                   = 335544884
	Isc_bad_teb_form                                              = 335544885
	Isc_tpb_multiple_txn_isolation                                = 335544886
	Isc_tpb_reserv_before_table                                   = 335544887
	Isc_tpb_multiple_spec                                         = 335544888
	Isc_tpb_option_without_rc                                     = 335544889
	Isc_tpb_conflicting_options                                   = 335544890
	Isc_tpb_reserv_missing_tlen                                   = 335544891
	Isc_tpb_reserv_long_tlen                                      = 335544892
	Isc_tpb_reserv_missing_tname                                  = 335544893
	Isc_tpb_reserv_corrup_tlen                                    = 335544894
	Isc_tpb_reserv_null_tlen                                      = 335544895
	Isc_tpb_reserv_relnotfound                                    = 335544896
	Isc_tpb_reserv_baserelnotfound                                = 335544897
	Isc_tpb_missing_len                                           = 335544898
	Isc_tpb_missing_value                                         = 335544899
	Isc_tpb_corrupt_len                                           = 335544900
	Isc_tpb_null_len                                              = 335544901
	Isc_tpb_overflow_len                                          = 335544902
	Isc_tpb_invalid_value                                         = 335544903
	Isc_tpb_reserv_stronger_wng                                   = 335544904
	Isc_tpb_reserv_stronger                                       = 335544905
	Isc_tpb_reserv_max_recursion                                  = 335544906
	Isc_tpb_reserv_virtualtbl                                     = 335544907
	Isc_tpb_reserv_systbl                                         = 335544908
	Isc_tpb_reserv_temptbl                                        = 335544909
	Isc_tpb_readtxn_after_writelock                               = 335544910
	Isc_tpb_writelock_after_readtxn                               = 335544911
	Isc_time_range_exceeded                                       = 335544912
	Isc_datetime_range_exceeded                                   = 335544913
	Isc_string_truncation                                         = 335544914
	Isc_blob_truncation                                           = 335544915
	Isc_numeric_out_of_range                                      = 335544916
	Isc_shutdown_timeout                                          = 335544917
	Isc_att_handle_busy                                           = 335544918
	Isc_bad_udf_freeit                                            = 335544919
	Isc_eds_provider_not_found                                    = 335544920
	Isc_eds_connection                                            = 335544921
	Isc_eds_preprocess                                            = 335544922
	Isc_eds_stmt_expected                                         = 335544923
	Isc_eds_prm_name_expected                                     = 335544924
	Isc_eds_unclosed_comment                                      = 335544925
	Isc_eds_statement                                             = 335544926
	Isc_eds_input_prm_mismatch                                    = 335544927
	Isc_eds_output_prm_mismatch                                   = 335544928
	Isc_eds_input_prm_not_set                                     = 335544929
	Isc_too_big_blr                                               = 335544930
	Isc_montabexh                                                 = 335544931
	Isc_modnotfound                                               = 335544932
	Isc_nothing_to_cancel                                         = 335544933
	Isc_ibutil_not_loaded                                         = 335544934
	Isc_circular_computed                                         = 335544935
	Isc_psw_db_error                                              = 335544936
	Isc_invalid_type_datetime_op                                  = 335544937
	Isc_onlycan_add_timetodate                                    = 335544938
	Isc_onlycan_add_datetotime                                    = 335544939
	Isc_onlycansub_tstampfromtstamp                               = 335544940
	Isc_onlyoneop_mustbe_tstamp                                   = 335544941
	Isc_invalid_extractpart_time                                  = 335544942
	Isc_invalid_extractpart_date                                  = 335544943
	Isc_invalidarg_extract                                        = 335544944
	Isc_sysf_argmustbe_exact                                      = 335544945
	Isc_sysf_argmustbe_exact_or_fp                                = 335544946
	Isc_sysf_argviolates_uuidtype                                 = 335544947
	Isc_sysf_argviolates_uuidlen                                  = 335544948
	Isc_sysf_argviolates_uuidfmt                                  = 335544949
	Isc_sysf_argviolates_guidigits                                = 335544950
	Isc_sysf_invalid_addpart_time                                 = 335544951
	Isc_sysf_invalid_add_datetime                                 = 335544952
	Isc_sysf_invalid_addpart_dtime                                = 335544953
	Isc_sysf_invalid_add_dtime_rc                                 = 335544954
	Isc_sysf_invalid_diff_dtime                                   = 335544955
	Isc_sysf_invalid_timediff                                     = 335544956
	Isc_sysf_invalid_tstamptimediff                               = 335544957
	Isc_sysf_invalid_datetimediff                                 = 335544958
	Isc_sysf_invalid_diffpart                                     = 335544959
	Isc_sysf_argmustbe_positive                                   = 335544960
	Isc_sysf_basemustbe_positive                                  = 335544961
	Isc_sysf_argnmustbe_nonneg                                    = 335544962
	Isc_sysf_argnmustbe_positive                                  = 335544963
	Isc_sysf_invalid_zeropowneg                                   = 335544964
	Isc_sysf_invalid_negpowfp                                     = 335544965
	Isc_sysf_invalid_scale                                        = 335544966
	Isc_sysf_argmustbe_nonneg                                     = 335544967
	Isc_sysf_binuuid_mustbe_str                                   = 335544968
	Isc_sysf_binuuid_wrongsize                                    = 335544969
	Isc_missing_required_spb                                      = 335544970
	Isc_net_server_shutdown                                       = 335544971
	Isc_bad_conn_str                                              = 335544972
	Isc_bad_epb_form                                              = 335544973
	Isc_no_threads                                                = 335544974
	Isc_net_event_connect_timeout                                 = 335544975
	Isc_sysf_argmustbe_nonzero                                    = 335544976
	Isc_sysf_argmustbe_range_inc1_1                               = 335544977
	Isc_sysf_argmustbe_gteq_one                                   = 335544978
	Isc_sysf_argmustbe_range_exc1_1                               = 335544979
	Isc_internal_rejected_params                                  = 335544980
	Isc_sysf_fp_overflow                                          = 335544981
	Isc_udf_fp_overflow                                           = 335544982
	Isc_udf_fp_nan                                                = 335544983
	Isc_instance_conflict                                         = 335544984
	Isc_out_of_temp_space                                         = 335544985
	Isc_eds_expl_tran_ctrl                                        = 335544986
	Isc_no_trusted_spb                                            = 335544987
	Isc_package_name                                              = 335544988
	Isc_cannot_make_not_null                                      = 335544989
	Isc_feature_removed                                           = 335544990
	Isc_view_name                                                 = 335544991
	Isc_lock_dir_access                                           = 335544992
	Isc_invalid_fetch_option                                      = 335544993
	Isc_bad_fun_BLR                                               = 335544994
	Isc_func_pack_not_implemented                                 = 335544995
	Isc_proc_pack_not_implemented                                 = 335544996
	Isc_eem_func_not_returned                                     = 335544997
	Isc_eem_proc_not_returned                                     = 335544998
	Isc_eem_trig_not_returned                                     = 335544999
	Isc_eem_bad_plugin_ver                                        = 335545000
	Isc_eem_engine_notfound                                       = 335545001
	Isc_attachment_in_use                                         = 335545002
	Isc_transaction_in_use                                        = 335545003
	Isc_pman_cannot_load_plugin                                   = 335545004
	Isc_pman_module_notfound                                      = 335545005
	Isc_pman_entrypoint_notfound                                  = 335545006
	Isc_pman_module_bad                                           = 335545007
	Isc_pman_plugin_notfound                                      = 335545008
	Isc_sysf_invalid_trig_namespace                               = 335545009
	Isc_unexpected_null                                           = 335545010
	Isc_type_notcompat_blob                                       = 335545011
	Isc_invalid_date_val                                          = 335545012
	Isc_invalid_time_val                                          = 335545013
	Isc_invalid_timestamp_val                                     = 335545014
	Isc_invalid_index_val                                         = 335545015
	Isc_formatted_exception                                       = 335545016
	Isc_async_active                                              = 335545017
	Isc_private_function                                          = 335545018
	Isc_private_procedure                                         = 335545019
	Isc_request_outdated                                          = 335545020
	Isc_bad_events_handle                                         = 335545021
	Isc_cannot_copy_stmt                                          = 335545022
	Isc_invalid_boolean_usage                                     = 335545023
	Isc_sysf_argscant_both_be_zero                                = 335545024
	Isc_spb_no_id                                                 = 335545025
	Isc_ee_blr_mismatch_null                                      = 335545026
	Isc_ee_blr_mismatch_length                                    = 335545027
	Isc_ss_out_of_bounds                                          = 335545028
	Isc_missing_data_structures                                   = 335545029
	Isc_protect_sys_tab                                           = 335545030
	Isc_libtommath_generic                                        = 335545031
	Isc_wroblrver2                                                = 335545032
	Isc_trunc_limits                                              = 335545033
	Isc_info_access                                               = 335545034
	Isc_svc_no_stdin                                              = 335545035
	Isc_svc_start_failed                                          = 335545036
	Isc_svc_no_switches                                           = 335545037
	Isc_svc_bad_size                                              = 335545038
	Isc_no_crypt_plugin                                           = 335545039
	Isc_cp_name_too_long                                          = 335545040
	Isc_cp_process_active                                         = 335545041
	Isc_cp_already_crypted                                        = 335545042
	Isc_decrypt_error                                             = 335545043
	Isc_no_providers                                              = 335545044
	Isc_null_spb                                                  = 335545045
	Isc_max_args_exceeded                                         = 335545046
	Isc_ee_blr_mismatch_names_count                               = 335545047
	Isc_ee_blr_mismatch_name_not_found                            = 335545048
	Isc_bad_result_set                                            = 335545049
	Isc_wrong_message_length                                      = 335545050
	Isc_no_output_format                                          = 335545051
	Isc_item_finish                                               = 335545052
	Isc_miss_config                                               = 335545053
	Isc_conf_line                                                 = 335545054
	Isc_conf_include                                              = 335545055
	Isc_include_depth                                             = 335545056
	Isc_include_miss                                              = 335545057
	Isc_protect_ownership                                         = 335545058
	Isc_badvarnum                                                 = 335545059
	Isc_sec_context                                               = 335545060
	Isc_multi_segment                                             = 335545061
	Isc_login_changed                                             = 335545062
	Isc_auth_handshake_limit                                      = 335545063
	Isc_wirecrypt_incompatible                                    = 335545064
	Isc_miss_wirecrypt                                            = 335545065
	Isc_wirecrypt_key                                             = 335545066
	Isc_wirecrypt_plugin                                          = 335545067
	Isc_secdb_name                                                = 335545068
	Isc_auth_data                                                 = 335545069
	Isc_auth_datalength                                           = 335545070
	Isc_info_unprepared_stmt                                      = 335545071
	Isc_idx_key_value                                             = 335545072
	Isc_forupdate_virtualtbl                                      = 335545073
	Isc_forupdate_systbl                                          = 335545074
	Isc_forupdate_temptbl                                         = 335545075
	Isc_cant_modify_sysobj                                        = 335545076
	Isc_server_misconfigured                                      = 335545077
	Isc_alter_role                                                = 335545078
	Isc_map_already_exists                                        = 335545079
	Isc_map_not_exists                                            = 335545080
	Isc_map_load                                                  = 335545081
	Isc_map_aster                                                 = 335545082
	Isc_map_multi                                                 = 335545083
	Isc_map_undefined                                             = 335545084
	Isc_baddpb_damaged_mode                                       = 335545085
	Isc_baddpb_buffers_range                                      = 335545086
	Isc_baddpb_temp_buffers                                       = 335545087
	Isc_map_nodb                                                  = 335545088
	Isc_map_notable                                               = 335545089
	Isc_miss_trusted_role                                         = 335545090
	Isc_set_invalid_role                                          = 335545091
	Isc_cursor_not_positioned                                     = 335545092
	Isc_dup_attribute                                             = 335545093
	Isc_dyn_no_priv                                               = 335545094
	Isc_dsql_cant_grant_option                                    = 335545095
	Isc_read_conflict                                             = 335545096
	Isc_crdb_load                                                 = 335545097
	Isc_crdb_nodb                                                 = 335545098
	Isc_crdb_notable                                              = 335545099
	Isc_interface_version_too_old                                 = 335545100
	Isc_fun_param_mismatch                                        = 335545101
	Isc_savepoint_backout_err                                     = 335545102
	Isc_domain_primary_key_notnull                                = 335545103
	Isc_invalid_attachment_charset                                = 335545104
	Isc_map_down                                                  = 335545105
	Isc_login_error                                               = 335545106
	Isc_already_opened                                            = 335545107
	Isc_bad_crypt_key                                             = 335545108
	Isc_encrypt_error                                             = 335545109
	Isc_max_idx_depth                                             = 335545110
	Isc_wrong_prvlg                                               = 335545111
	Isc_miss_prvlg                                                = 335545112
	Isc_crypt_checksum                                            = 335545113
	Isc_not_dba                                                   = 335545114
	Isc_no_cursor                                                 = 335545115
	Isc_dsql_window_incompat_frames                               = 335545116
	Isc_dsql_window_range_multi_key                               = 335545117
	Isc_dsql_window_range_inv_key_type                            = 335545118
	Isc_dsql_window_frame_value_inv_type                          = 335545119
	Isc_window_frame_value_invalid                                = 335545120
	Isc_dsql_window_not_found                                     = 335545121
	Isc_dsql_window_cant_overr_part                               = 335545122
	Isc_dsql_window_cant_overr_order                              = 335545123
	Isc_dsql_window_cant_overr_frame                              = 335545124
	Isc_dsql_window_duplicate                                     = 335545125
	Isc_sql_too_long                                              = 335545126
	Isc_cfg_stmt_timeout                                          = 335545127
	Isc_att_stmt_timeout                                          = 335545128
	Isc_req_stmt_timeout                                          = 335545129
	Isc_att_shut_killed                                           = 335545130
	Isc_att_shut_idle                                             = 335545131
	Isc_att_shut_db_down                                          = 335545132
	Isc_att_shut_engine                                           = 335545133
	Isc_overriding_without_identity                               = 335545134
	Isc_overriding_system_invalid                                 = 335545135
	Isc_overriding_user_invalid                                   = 335545136
	Isc_overriding_missing                                        = 335545137
	Isc_decprecision_err                                          = 335545138
	Isc_decfloat_divide_by_zero                                   = 335545139
	Isc_decfloat_inexact_result                                   = 335545140
	Isc_decfloat_invalid_operation                                = 335545141
	Isc_decfloat_overflow                                         = 335545142
	Isc_decfloat_underflow                                        = 335545143
	Isc_subfunc_notdef                                            = 335545144
	Isc_subproc_notdef                                            = 335545145
	Isc_subfunc_signat                                            = 335545146
	Isc_subproc_signat                                            = 335545147
	Isc_subfunc_defvaldecl                                        = 335545148
	Isc_subproc_defvaldecl                                        = 335545149
	Isc_subfunc_not_impl                                          = 335545150
	Isc_subproc_not_impl                                          = 335545151
	Isc_sysf_invalid_hash_algorithm                               = 335545152
	Isc_expression_eval_index                                     = 335545153
	Isc_invalid_decfloat_trap                                     = 335545154
	Isc_invalid_decfloat_round                                    = 335545155
	Isc_sysf_invalid_first_last_part                              = 335545156
	Isc_sysf_invalid_date_timestamp                               = 335545157
	Isc_precision_err2                                            = 335545158
	Isc_bad_batch_handle                                          = 335545159
	Isc_intl_char                                                 = 335545160
	Isc_null_block                                                = 335545161
	Isc_mixed_info                                                = 335545162
	Isc_unknown_info                                              = 335545163
	Isc_bpb_version                                               = 335545164
	Isc_user_manager                                              = 335545165
	Isc_icu_entrypoint                                            = 335545166
	Isc_icu_library                                               = 335545167
	Isc_metadata_name                                             = 335545168
	Isc_tokens_parse                                              = 335545169
	Isc_iconv_open                                                = 335545170
	Isc_batch_compl_range                                         = 335545171
	Isc_batch_compl_detail                                        = 335545172
	Isc_deflate_init                                              = 335545173
	Isc_inflate_init                                              = 335545174
	Isc_big_segment                                               = 335545175
	Isc_batch_policy                                              = 335545176
	Isc_batch_defbpb                                              = 335545177
	Isc_batch_align                                               = 335545178
	Isc_multi_segment_dup                                         = 335545179
	Isc_non_plugin_protocol                                       = 335545180
	Isc_message_format                                            = 335545181
	Isc_batch_param_version                                       = 335545182
	Isc_batch_msg_long                                            = 335545183
	Isc_batch_open                                                = 335545184
	Isc_batch_type                                                = 335545185
	Isc_batch_param                                               = 335545186
	Isc_batch_blobs                                               = 335545187
	Isc_batch_blob_append                                         = 335545188
	Isc_batch_stream_align                                        = 335545189
	Isc_batch_rpt_blob                                            = 335545190
	Isc_batch_blob_buf                                            = 335545191
	Isc_batch_small_data                                          = 335545192
	Isc_batch_cont_bpb                                            = 335545193
	Isc_batch_big_bpb                                             = 335545194
	Isc_batch_big_segment                                         = 335545195
	Isc_batch_big_seg2                                            = 335545196
	Isc_batch_blob_id                                             = 335545197
	Isc_batch_too_big                                             = 335545198
	Isc_num_literal                                               = 335545199
	Isc_map_event                                                 = 335545200
	Isc_map_overflow                                              = 335545201
	Isc_hdr_overflow                                              = 335545202
	Isc_vld_plugins                                               = 335545203
	Isc_db_crypt_key                                              = 335545204
	Isc_no_keyholder_plugin                                       = 335545205
	Isc_ses_reset_err                                             = 335545206
	Isc_ses_reset_open_trans                                      = 335545207
	Isc_ses_reset_warn                                            = 335545208
	Isc_ses_reset_tran_rollback                                   = 335545209
	Isc_plugin_name                                               = 335545210
	Isc_parameter_name                                            = 335545211
	Isc_file_starting_page_err                                    = 335545212
	Isc_invalid_timezone_offset                                   = 335545213
	Isc_invalid_timezone_region                                   = 335545214
	Isc_invalid_timezone_id                                       = 335545215
	Isc_tom_decode64len                                           = 335545216
	Isc_tom_strblob                                               = 335545217
	Isc_tom_reg                                                   = 335545218
	Isc_tom_algorithm                                             = 335545219
	Isc_tom_mode_miss                                             = 335545220
	Isc_tom_mode_bad                                              = 335545221
	Isc_tom_no_mode                                               = 335545222
	Isc_tom_iv_miss                                               = 335545223
	Isc_tom_no_iv                                                 = 335545224
	Isc_tom_ctrtype_bad                                           = 335545225
	Isc_tom_no_ctrtype                                            = 335545226
	Isc_tom_ctr_big                                               = 335545227
	Isc_tom_no_ctr                                                = 335545228
	Isc_tom_iv_length                                             = 335545229
	Isc_tom_error                                                 = 335545230
	Isc_tom_yarrow_start                                          = 335545231
	Isc_tom_yarrow_setup                                          = 335545232
	Isc_tom_init_mode                                             = 335545233
	Isc_tom_crypt_mode                                            = 335545234
	Isc_tom_decrypt_mode                                          = 335545235
	Isc_tom_init_cip                                              = 335545236
	Isc_tom_crypt_cip                                             = 335545237
	Isc_tom_decrypt_cip                                           = 335545238
	Isc_tom_setup_cip                                             = 335545239
	Isc_tom_setup_chacha                                          = 335545240
	Isc_tom_encode                                                = 335545241
	Isc_tom_decode                                                = 335545242
	Isc_tom_rsa_import                                            = 335545243
	Isc_tom_oaep                                                  = 335545244
	Isc_tom_hash_bad                                              = 335545245
	Isc_tom_rsa_make                                              = 335545246
	Isc_tom_rsa_export                                            = 335545247
	Isc_tom_rsa_sign                                              = 335545248
	Isc_tom_rsa_verify                                            = 335545249
	Isc_tom_chacha_key                                            = 335545250
	Isc_bad_repl_handle                                           = 335545251
	Isc_tra_snapshot_does_not_exist                               = 335545252
	Isc_eds_input_prm_not_used                                    = 335545253
	Isc_effective_user                                            = 335545254
	Isc_invalid_time_zone_bind                                    = 335545255
	Isc_invalid_decfloat_bind                                     = 335545256
	Isc_odd_hex_len                                               = 335545257
	Isc_invalid_hex_digit                                         = 335545258
	Isc_bind_err                                                  = 335545259
	Isc_bind_statement                                            = 335545260
	Isc_bind_convert                                              = 335545261
	Isc_cannot_update_old_blob                                    = 335545262
	Isc_cannot_read_new_blob                                      = 335545263
	Isc_dyn_no_create_priv                                        = 335545264
	Isc_suspend_without_returns                                   = 335545265
	Isc_truncate_warn                                             = 335545266
	Isc_truncate_monitor                                          = 335545267
	Isc_truncate_context                                          = 335545268
	Isc_merge_dup_update                                          = 335545269
	Isc_wrong_page                                                = 335545270
	Isc_repl_error                                                = 335545271
	Isc_ses_reset_failed                                          = 335545272
	Isc_block_size                                                = 335545273
	Isc_tom_key_length                                            = 335545274
	Isc_inf_invalid_args                                          = 335545275
	Isc_sysf_invalid_null_empty                                   = 335545276
	Isc_bad_loctab_num                                            = 335545277
	Isc_quoted_str_bad                                            = 335545278
	Isc_quoted_str_miss                                           = 335545279
	Isc_wrong_shmem_ver                                           = 335545280
	Isc_wrong_shmem_bitness                                       = 335545281
	Isc_wrong_proc_plan                                           = 335545282
	Isc_invalid_blob_util_handle                                  = 335545283
	Isc_bad_temp_blob_id                                          = 335545284
	Isc_ods_upgrade_err                                           = 335545285
	Isc_bad_par_workers                                           = 335545286
	Isc_idx_expr_not_found                                        = 335545287
	Isc_idx_cond_not_found                                        = 335545288
	Isc_uninitialized_var                                         = 335545289
	Isc_param_not_exist                                           = 335545290
	Isc_param_no_default_not_specified                            = 335545291
	Isc_param_multiple_assignments                                = 335545292
	Isc_invalid_date_format                                       = 335545293
	Isc_invalid_raw_string_in_date_format                         = 335545294
	Isc_invalid_data_type_for_date_format                         = 335545295
	Isc_incompatible_date_format_with_current_date_type           = 335545296
	Isc_value_for_pattern_is_out_of_range                         = 335545297
	Isc_month_name_mismatch                                       = 335545298
	Isc_incorrect_hours_period                                    = 335545299
	Isc_data_for_format_is_exhausted                              = 335545300
	Isc_trailing_part_of_string                                   = 335545301
	Isc_pattern_cant_be_used_without_other_pattern                = 335545302
	Isc_pattern_cant_be_used_without_other_pattern_and_vice_versa = 335545303
	Isc_incompatible_format_patterns                              = 335545304
	Isc_only_one_pattern_can_be_used                              = 335545305
	Isc_can_not_use_same_pattern_twice                            = 335545306
	Isc_sysf_invalid_gen_uuid_version                             = 335545307
	Isc_sweep_unable_to_run                                       = 335545308
	Isc_sweep_concurrent_instance                                 = 335545309
	Isc_sweep_read_only                                           = 335545310
	Isc_sweep_attach_no_cleanup                                   = 335545311
	Isc_invalid_timezone_region_or_displacement                   = 335545312
	Isc_argmustbe_exact_range_for                                 = 335545313
	Isc_range_for_by_should_be_positive                           = 335545314
	Isc_missing_value_for_format_pattern                          = 335545315
	Isc_invalid_name                                              = 335545316
	Isc_invalid_unqualified_name_list                             = 335545317
	Isc_no_user_att_while_restore                                 = 335545318
	Isc_genseq_stepmustbe_nonzero                                 = 335545319
	Isc_argmustbe_exact_function                                  = 335545320
	Isc_sysf_argmustbe_range_inc0_1                               = 335545321
	Isc_argmustbe_numeric_function                                = 335545322
	Isc_percetile_only_one_sort_item                              = 335545323
	Isc_argmustbe_const_within_group                              = 335545324
	Isc_update_overwrite                                          = 335545325
	Isc_pman_plugin_dirname                                       = 335545326
	Isc_private_constant                                          = 335545327
	Isc_package_alias_help                                        = 335545328
	Isc_bad_constant_blr                                          = 335545329
	Isc_bad_constant_desc                                         = 335545330
	Isc_bad_constant_name                                         = 335545331
	Isc_bad_constant_type                                         = 335545332
	Isc_not_defined_constant                                      = 335545333
	Isc_const_name                                                = 335545334
	Isc_private_table                                             = 335545335
	Isc_temp_space_invalid_pos                                    = 335545336
	Isc_dsql_agg_non_agg_context                                  = 335545337
	Isc_dsql_agg_param_not_accum                                  = 335545338
	Isc_dsql_agg_exit_group                                       = 335545339
	Isc_dsql_agg_return                                           = 335545340
	Isc_bad_ext_file                                              = 335546320
	Isc_wrong_adp_fields_def                                      = 335546321
	Isc_wrong_adp_field_name                                      = 335546322
	Isc_wrong_adp_field_type                                      = 335546323
	Isc_bad_ext_record                                            = 335546324
	Isc_bad_adp_type                                              = 335546325
	Isc_dir_resolve_error                                         = 335546326
	Isc_read_file_error                                           = 335546327
	Isc_create_file_error                                         = 335546328
	Isc_delete_file_error                                         = 335546329
	Isc_blob_io_error                                             = 335546330
	Isc_cant_connect_to_ldap                                      = 335546331
	Isc_cant_bind_to_ldap                                         = 335546332
	Isc_no_ldap_init                                              = 335546333
	Isc_sysf_ldap_attr                                            = 335546334
	Isc_no_ext_file                                               = 335546335
	Isc_no_gss_init                                               = 335546336
	Isc_hash_error                                                = 335546337
	Isc_policy_already_exists                                     = 335546338
	Isc_policy_not_exists                                         = 335546339
	Isc_user_not_exists                                           = 335546340
	Isc_policy_is_granted                                         = 335546341
	Isc_bad_trig_BLR                                              = 335546342
	Isc_not_rdb                                                   = 335546343
	Isc_trace_write_error                                         = 335546345
	Isc_no_granted_policy                                         = 335546346
	Isc_regexp_substr_error                                       = 335546347
	Isc_secdb_coonection_error                                    = 335546351
	Isc_policies_reading_error                                    = 335546352
	Isc_get_policies_error                                        = 335546353
	Isc_att_suspended                                             = 335546354
	Isc_suspend_privileges_error                                  = 335546355
	Isc_trace_buffer_limit                                        = 335546356
	Isc_pman_hash_incorrect                                       = 335546357
	Isc_json_table_plan_missing_name                              = 335546358
	Isc_json_table_same_names                                     = 335546359
	Isc_json_table_plan_same_names                                = 335546360
	Isc_json_table_plan_double_appearance                         = 335546361
	Isc_json_table_plan_wrong_level                               = 335546362
	Isc_json_table_invalid_plan                                   = 335546363
	Isc_json_table_plan_wrong_name                                = 335546364
	Isc_foreign_server                                            = 335546369
	Isc_foreign_provider_not_found                                = 335546370
	Isc_change_forced_write_mode_with_connections                 = 335546372
	Isc_license_not_found                                         = 335546373
	Isc_feature_not_available                                     = 335546374
	Isc_repl_mispos                                               = 335546375
	Isc_missing_trace_plugin                                      = 335546378
	Isc_repl_recover_err                                          = 335546379
	Isc_user_policy_reading_error                                 = 335546380
	Isc_get_user_policy_error                                     = 335546381
	Isc_gfix_db_name                                              = 335740929
	Isc_gfix_invalid_sw                                           = 335740930
	Isc_gfix_version                                              = 335740931
	Isc_gfix_incmp_sw                                             = 335740932
	Isc_gfix_replay_req                                           = 335740933
	Isc_gfix_pgbuf_req                                            = 335740934
	Isc_gfix_val_req                                              = 335740935
	Isc_gfix_pval_req                                             = 335740936
	Isc_gfix_trn_req                                              = 335740937
	Isc_gfix_trn_all_req                                          = 335740938
	Isc_gfix_sync_req                                             = 335740939
	Isc_gfix_full_req                                             = 335740940
	Isc_gfix_usrname_req                                          = 335740941
	Isc_gfix_pass_req                                             = 335740942
	Isc_gfix_subs_name                                            = 335740943
	Isc_gfix_wal_req                                              = 335740944
	Isc_gfix_sec_req                                              = 335740945
	Isc_gfix_nval_req                                             = 335740946
	Isc_gfix_type_shut                                            = 335740947
	Isc_gfix_retry                                                = 335740948
	Isc_gfix_opt                                                  = 335740949
	Isc_gfix_qualifiers                                           = 335740950
	Isc_gfix_retry_db                                             = 335740951
	Isc_gfix_summary                                              = 335740952
	Isc_gfix_opt_active                                           = 335740953
	Isc_gfix_opt_attach                                           = 335740954
	Isc_gfix_opt_begin_log                                        = 335740955
	Isc_gfix_opt_buffers                                          = 335740956
	Isc_gfix_opt_commit                                           = 335740957
	Isc_gfix_opt_cache                                            = 335740958
	Isc_gfix_opt_disable                                          = 335740959
	Isc_gfix_opt_full                                             = 335740960
	Isc_gfix_opt_force                                            = 335740961
	Isc_gfix_opt_housekeep                                        = 335740962
	Isc_gfix_opt_ignore                                           = 335740963
	Isc_gfix_opt_kill                                             = 335740964
	Isc_gfix_opt_list                                             = 335740965
	Isc_gfix_opt_mend                                             = 335740966
	Isc_gfix_opt_no_update                                        = 335740967
	Isc_gfix_opt_online                                           = 335740968
	Isc_gfix_opt_prompt                                           = 335740969
	Isc_gfix_opt_password                                         = 335740970
	Isc_gfix_opt_quit_log                                         = 335740971
	Isc_gfix_opt_rollback                                         = 335740972
	Isc_gfix_opt_sweep                                            = 335740973
	Isc_gfix_opt_shut                                             = 335740974
	Isc_gfix_opt_two_phase                                        = 335740975
	Isc_gfix_opt_tran                                             = 335740976
	Isc_gfix_opt_use                                              = 335740977
	Isc_gfix_opt_user                                             = 335740978
	Isc_gfix_opt_validate                                         = 335740979
	Isc_gfix_opt_write                                            = 335740980
	Isc_gfix_opt_x                                                = 335740981
	Isc_gfix_opt_z                                                = 335740982
	Isc_gfix_rec_err                                              = 335740983
	Isc_gfix_blob_err                                             = 335740984
	Isc_gfix_data_err                                             = 335740985
	Isc_gfix_index_err                                            = 335740986
	Isc_gfix_pointer_err                                          = 335740987
	Isc_gfix_trn_err                                              = 335740988
	Isc_gfix_db_err                                               = 335740989
	Isc_gfix_bad_block                                            = 335740990
	Isc_gfix_exceed_max                                           = 335740991
	Isc_gfix_corrupt_pool                                         = 335740992
	Isc_gfix_mem_exhausted                                        = 335740993
	Isc_gfix_bad_pool                                             = 335740994
	Isc_gfix_trn_not_valid                                        = 335740995
	Isc_gfix_dbg_attach                                           = 335740996
	Isc_gfix_dbg_failed                                           = 335740997
	Isc_gfix_dbg_success                                          = 335740998
	Isc_gfix_trn_limbo                                            = 335740999
	Isc_gfix_try_again                                            = 335741000
	Isc_gfix_unrec_item                                           = 335741001
	Isc_gfix_commit_violate                                       = 335741002
	Isc_gfix_preserve                                             = 335741003
	Isc_gfix_part_commit                                          = 335741004
	Isc_gfix_rback_violate                                        = 335741005
	Isc_gfix_part_commit2                                         = 335741006
	Isc_gfix_commit_pres                                          = 335741007
	Isc_gfix_insuff_info                                          = 335741008
	Isc_gfix_action                                               = 335741009
	Isc_gfix_all_prep                                             = 335741010
	Isc_gfix_comm_rback                                           = 335741011
	Isc_gfix_unexp_eoi                                            = 335741012
	Isc_gfix_ask                                                  = 335741013
	Isc_gfix_reattach_failed                                      = 335741014
	Isc_gfix_org_path                                             = 335741015
	Isc_gfix_enter_path                                           = 335741016
	Isc_gfix_att_unsucc                                           = 335741017
	Isc_gfix_recon_fail                                           = 335741018
	Isc_gfix_trn2                                                 = 335741019
	Isc_gfix_mdb_trn                                              = 335741020
	Isc_gfix_host_site                                            = 335741021
	Isc_gfix_trn                                                  = 335741022
	Isc_gfix_prepared                                             = 335741023
	Isc_gfix_committed                                            = 335741024
	Isc_gfix_rolled_back                                          = 335741025
	Isc_gfix_not_available                                        = 335741026
	Isc_gfix_not_prepared                                         = 335741027
	Isc_gfix_be_committed                                         = 335741028
	Isc_gfix_rmt_site                                             = 335741029
	Isc_gfix_db_path                                              = 335741030
	Isc_gfix_auto_comm                                            = 335741031
	Isc_gfix_auto_rback                                           = 335741032
	Isc_gfix_warning                                              = 335741033
	Isc_gfix_trn_was_comm                                         = 335741034
	Isc_gfix_trn_was_rback                                        = 335741035
	Isc_gfix_trn_unknown                                          = 335741036
	Isc_gfix_opt_mode                                             = 335741037
	Isc_gfix_mode_req                                             = 335741038
	Isc_gfix_opt_SQL_dialect                                      = 335741039
	Isc_gfix_SQL_dialect                                          = 335741040
	Isc_gfix_dialect_req                                          = 335741041
	Isc_gfix_pzval_req                                            = 335741042
	Isc_gfix_opt_trusted                                          = 335741043
	Isc_gfix_opt_nolinger                                         = 335741049
	Isc_gfix_pip_err                                              = 335741050
	Isc_gfix_rec_warn                                             = 335741051
	Isc_gfix_blob_warn                                            = 335741052
	Isc_gfix_data_warn                                            = 335741053
	Isc_gfix_index_warn                                           = 335741054
	Isc_gfix_pointer_warn                                         = 335741055
	Isc_gfix_trn_warn                                             = 335741056
	Isc_gfix_db_warn                                              = 335741057
	Isc_gfix_pip_warn                                             = 335741058
	Isc_gfix_opt_icu                                              = 335741059
	Isc_gfix_opt_role                                             = 335741060
	Isc_gfix_role_req                                             = 335741061
	Isc_gfix_opt_repl                                             = 335741062
	Isc_gfix_repl_mode_req                                        = 335741063
	Isc_gfix_opt_parallel                                         = 335741064
	Isc_gfix_opt_upgrade                                          = 335741065
	Isc_gfix_overval_req                                          = 335741066
	Isc_gfix_opt_sweep_timeout                                    = 335741067
	Isc_gfix_opt_guid                                             = 335741928
	Isc_gfix_opt_config_val                                       = 335741931
	Isc_dsql_dbkey_from_non_table                                 = 336003074
	Isc_dsql_transitional_numeric                                 = 336003075
	Isc_dsql_dialect_warning_expr                                 = 336003076
	Isc_sql_db_dialect_dtype_unsupport                            = 336003077
	Isc_sql_dialect_conflict_num                                  = 336003079
	Isc_dsql_warning_number_ambiguous                             = 336003080
	Isc_dsql_warning_number_ambiguous1                            = 336003081
	Isc_dsql_warn_precision_ambiguous                             = 336003082
	Isc_dsql_warn_precision_ambiguous1                            = 336003083
	Isc_dsql_warn_precision_ambiguous2                            = 336003084
	Isc_dsql_ambiguous_field_name                                 = 336003085
	Isc_dsql_udf_return_pos_err                                   = 336003086
	Isc_dsql_invalid_label                                        = 336003087
	Isc_dsql_datatypes_not_comparable                             = 336003088
	Isc_dsql_cursor_invalid                                       = 336003089
	Isc_dsql_cursor_redefined                                     = 336003090
	Isc_dsql_cursor_not_found                                     = 336003091
	Isc_dsql_cursor_exists                                        = 336003092
	Isc_dsql_cursor_rel_ambiguous                                 = 336003093
	Isc_dsql_cursor_rel_not_found                                 = 336003094
	Isc_dsql_cursor_not_open                                      = 336003095
	Isc_dsql_type_not_supp_ext_tab                                = 336003096
	Isc_dsql_feature_not_supported_ods                            = 336003097
	Isc_primary_key_required                                      = 336003098
	Isc_upd_ins_doesnt_match_pk                                   = 336003099
	Isc_upd_ins_doesnt_match_matching                             = 336003100
	Isc_upd_ins_with_complex_view                                 = 336003101
	Isc_dsql_incompatible_trigger_type                            = 336003102
	Isc_dsql_db_trigger_type_cant_change                          = 336003103
	Isc_dsql_record_version_table                                 = 336003104
	Isc_dsql_invalid_sqlda_version                                = 336003105
	Isc_dsql_sqlvar_index                                         = 336003106
	Isc_dsql_no_sqlind                                            = 336003107
	Isc_dsql_no_sqldata                                           = 336003108
	Isc_dsql_no_input_sqlda                                       = 336003109
	Isc_dsql_no_output_sqlda                                      = 336003110
	Isc_dsql_wrong_param_num                                      = 336003111
	Isc_dsql_invalid_drop_ss_clause                               = 336003112
	Isc_upd_ins_cannot_default                                    = 336003113
	Isc_dsql_ltt_invalid_reference                                = 336003114
	Isc_dsql_using_statement_must_contain_clause                  = 336003115
	Isc_dyn_filter_not_found                                      = 336068645
	Isc_dyn_func_not_found                                        = 336068649
	Isc_dyn_index_not_found                                       = 336068656
	Isc_dyn_view_not_found                                        = 336068662
	Isc_dyn_domain_not_found                                      = 336068697
	Isc_dyn_cant_modify_auto_trig                                 = 336068717
	Isc_dyn_dup_table                                             = 336068740
	Isc_dyn_dup_procedure                                         = 336068743
	Isc_dyn_proc_not_found                                        = 336068748
	Isc_dyn_exception_not_found                                   = 336068752
	Isc_dyn_proc_param_not_found                                  = 336068754
	Isc_dyn_trig_not_found                                        = 336068755
	Isc_dyn_charset_not_found                                     = 336068759
	Isc_dyn_collation_not_found                                   = 336068760
	Isc_dyn_role_not_found                                        = 336068763
	Isc_dyn_name_longer                                           = 336068767
	Isc_dyn_column_does_not_exist                                 = 336068784
	Isc_dyn_role_does_not_exist                                   = 336068796
	Isc_dyn_no_grant_admin_opt                                    = 336068797
	Isc_dyn_user_not_role_member                                  = 336068798
	Isc_dyn_delete_role_failed                                    = 336068799
	Isc_dyn_grant_role_to_user                                    = 336068800
	Isc_dyn_inv_sql_role_name                                     = 336068801
	Isc_dyn_dup_sql_role                                          = 336068802
	Isc_dyn_kywd_spec_for_role                                    = 336068803
	Isc_dyn_roles_not_supported                                   = 336068804
	Isc_dyn_domain_name_exists                                    = 336068812
	Isc_dyn_field_name_exists                                     = 336068813
	Isc_dyn_dependency_exists                                     = 336068814
	Isc_dyn_dtype_invalid                                         = 336068815
	Isc_dyn_char_fld_too_small                                    = 336068816
	Isc_dyn_invalid_dtype_conversion                              = 336068817
	Isc_dyn_dtype_conv_invalid                                    = 336068818
	Isc_dyn_virmemexh                                             = 336068819
	Isc_dyn_zero_len_id                                           = 336068820
	Isc_del_gen_fail                                              = 336068821
	Isc_dyn_gen_not_found                                         = 336068822
	Isc_max_coll_per_charset                                      = 336068829
	Isc_invalid_coll_attr                                         = 336068830
	Isc_dyn_wrong_gtt_scope                                       = 336068840
	Isc_del_coll_fail                                             = 336068842
	Isc_dyn_coll_used_table                                       = 336068843
	Isc_dyn_coll_used_domain                                      = 336068844
	Isc_dyn_cannot_del_syscoll                                    = 336068845
	Isc_dyn_cannot_del_def_coll                                   = 336068846
	Isc_dyn_table_not_found                                       = 336068849
	Isc_dyn_coll_used_procedure                                   = 336068851
	Isc_dyn_scale_too_big                                         = 336068852
	Isc_dyn_precision_too_small                                   = 336068853
	Isc_dyn_miss_priv_warning                                     = 336068855
	Isc_dyn_ods_not_supp_feature                                  = 336068856
	Isc_dyn_cannot_addrem_computed                                = 336068857
	Isc_dyn_no_empty_pw                                           = 336068858
	Isc_dyn_dup_index                                             = 336068859
	Isc_dyn_locksmith_use_granted                                 = 336068860
	Isc_dyn_dup_exception                                         = 336068861
	Isc_dyn_dup_generator                                         = 336068862
	Isc_dyn_package_not_found                                     = 336068864
	Isc_dyn_schema_not_found                                      = 336068865
	Isc_dyn_cannot_mod_sysproc                                    = 336068866
	Isc_dyn_cannot_mod_systrig                                    = 336068867
	Isc_dyn_cannot_mod_sysfunc                                    = 336068868
	Isc_dyn_invalid_ddl_proc                                      = 336068869
	Isc_dyn_invalid_ddl_trig                                      = 336068870
	Isc_dyn_funcnotdef_package                                    = 336068871
	Isc_dyn_procnotdef_package                                    = 336068872
	Isc_dyn_funcsignat_package                                    = 336068873
	Isc_dyn_procsignat_package                                    = 336068874
	Isc_dyn_defvaldecl_package_proc                               = 336068875
	Isc_dyn_dup_function                                          = 336068876
	Isc_dyn_package_body_exists                                   = 336068877
	Isc_dyn_invalid_ddl_func                                      = 336068878
	Isc_dyn_newfc_oldsyntax                                       = 336068879
	Isc_dyn_func_param_not_found                                  = 336068886
	Isc_dyn_routine_param_not_found                               = 336068887
	Isc_dyn_routine_param_ambiguous                               = 336068888
	Isc_dyn_coll_used_function                                    = 336068889
	Isc_dyn_domain_used_function                                  = 336068890
	Isc_dyn_alter_user_no_clause                                  = 336068891
	Isc_dyn_duplicate_package_item                                = 336068894
	Isc_dyn_cant_modify_sysobj                                    = 336068895
	Isc_dyn_cant_use_zero_increment                               = 336068896
	Isc_dyn_cant_use_in_foreignkey                                = 336068897
	Isc_dyn_defvaldecl_package_func                               = 336068898
	Isc_dyn_create_user_no_password                               = 336068899
	Isc_dyn_cyclic_role                                           = 336068900
	Isc_dyn_cant_use_zero_inc_ident                               = 336068904
	Isc_dyn_concur_alter_database                                 = 336068905
	Isc_dyn_incompat_alter_database                               = 336068906
	Isc_dyn_no_ddl_grant_opt_priv                                 = 336068907
	Isc_dyn_no_grant_opt_priv                                     = 336068908
	Isc_dyn_func_not_exist                                        = 336068909
	Isc_dyn_proc_not_exist                                        = 336068910
	Isc_dyn_pack_not_exist                                        = 336068911
	Isc_dyn_trig_not_exist                                        = 336068912
	Isc_dyn_view_not_exist                                        = 336068913
	Isc_dyn_rel_not_exist                                         = 336068914
	Isc_dyn_exc_not_exist                                         = 336068915
	Isc_dyn_gen_not_exist                                         = 336068916
	Isc_dyn_fld_not_exist                                         = 336068917
	Isc_dyn_dup_trigger                                           = 336068918
	Isc_dyn_dup_domain                                            = 336068919
	Isc_dyn_dup_collation                                         = 336068920
	Isc_dyn_dup_package                                           = 336068921
	Isc_dyn_index_schema_must_match_table                         = 336068922
	Isc_dyn_trig_schema_must_match_table                          = 336068923
	Isc_dyn_dup_schema                                            = 336068924
	Isc_dyn_cannot_mod_system_schema                              = 336068925
	Isc_dyn_cannot_drop_non_emptyschema                           = 336068926
	Isc_dyn_cannot_mod_obj_sys_schema                             = 336068927
	Isc_dyn_cannot_create_reserved_schema                         = 336068928
	Isc_dyn_cannot_infer_schema                                   = 336068929
	Isc_dyn_dup_blob_filter                                       = 336068930
	Isc_dyn_column_name_exists                                    = 336068931
	Isc_dyn_constant_not_found                                    = 336068932
	Isc_dyn_dup_const                                             = 336068933
	Isc_dyn_non_constant_constant                                 = 336068934
	Isc_dyn_function_mismatch                                     = 336068935
	Isc_dyn_dup_domain_rdb                                        = 336069609
	Isc_dyn_dup_trigger_rdb                                       = 336069610
	Isc_dyn_dup_package_rdb                                       = 336069611
	Isc_dyn_dup_filter                                            = 336069612
	Isc_dyn_job_not_found                                         = 336069613
	Isc_dyn_invalid_ddl_job                                       = 336069614
	Isc_dyn_alter_job_no_clause                                   = 336069615
	Isc_dyn_cant_modify_collation                                 = 336069618
	Isc_dyn_foreign_server_not_found                              = 336069620
	Isc_dyn_user_mapping_not_found                                = 336069621
	Isc_dyn_job_already_exists                                    = 336069622
	Isc_dyn_foreign_server_already_exists                         = 336069624
	Isc_gbak_unknown_switch                                       = 336330753
	Isc_gbak_page_size_missing                                    = 336330754
	Isc_gbak_page_size_toobig                                     = 336330755
	Isc_gbak_redir_ouput_missing                                  = 336330756
	Isc_gbak_switches_conflict                                    = 336330757
	Isc_gbak_unknown_device                                       = 336330758
	Isc_gbak_no_protection                                        = 336330759
	Isc_gbak_page_size_not_allowed                                = 336330760
	Isc_gbak_multi_source_dest                                    = 336330761
	Isc_gbak_filename_missing                                     = 336330762
	Isc_gbak_dup_inout_names                                      = 336330763
	Isc_gbak_inv_page_size                                        = 336330764
	Isc_gbak_db_specified                                         = 336330765
	Isc_gbak_db_exists                                            = 336330766
	Isc_gbak_unk_device                                           = 336330767
	Isc_gbak_blob_info_failed                                     = 336330772
	Isc_gbak_unk_blob_item                                        = 336330773
	Isc_gbak_get_seg_failed                                       = 336330774
	Isc_gbak_close_blob_failed                                    = 336330775
	Isc_gbak_open_blob_failed                                     = 336330776
	Isc_gbak_put_blr_gen_id_failed                                = 336330777
	Isc_gbak_unk_type                                             = 336330778
	Isc_gbak_comp_req_failed                                      = 336330779
	Isc_gbak_start_req_failed                                     = 336330780
	Isc_gbak_rec_failed                                           = 336330781
	Isc_gbak_rel_req_failed                                       = 336330782
	Isc_gbak_db_info_failed                                       = 336330783
	Isc_gbak_no_db_desc                                           = 336330784
	Isc_gbak_db_create_failed                                     = 336330785
	Isc_gbak_decomp_len_error                                     = 336330786
	Isc_gbak_tbl_missing                                          = 336330787
	Isc_gbak_blob_col_missing                                     = 336330788
	Isc_gbak_create_blob_failed                                   = 336330789
	Isc_gbak_put_seg_failed                                       = 336330790
	Isc_gbak_rec_len_exp                                          = 336330791
	Isc_gbak_inv_rec_len                                          = 336330792
	Isc_gbak_exp_data_type                                        = 336330793
	Isc_gbak_gen_id_failed                                        = 336330794
	Isc_gbak_unk_rec_type                                         = 336330795
	Isc_gbak_inv_bkup_ver                                         = 336330796
	Isc_gbak_missing_bkup_desc                                    = 336330797
	Isc_gbak_string_trunc                                         = 336330798
	Isc_gbak_cant_rest_record                                     = 336330799
	Isc_gbak_send_failed                                          = 336330800
	Isc_gbak_no_tbl_name                                          = 336330801
	Isc_gbak_unexp_eof                                            = 336330802
	Isc_gbak_db_format_too_old                                    = 336330803
	Isc_gbak_inv_array_dim                                        = 336330804
	Isc_gbak_xdr_len_expected                                     = 336330807
	Isc_gbak_open_bkup_error                                      = 336330817
	Isc_gbak_open_error                                           = 336330818
	Isc_gbak_missing_block_fac                                    = 336330934
	Isc_gbak_inv_block_fac                                        = 336330935
	Isc_gbak_block_fac_specified                                  = 336330936
	Isc_gbak_missing_username                                     = 336330940
	Isc_gbak_missing_password                                     = 336330941
	Isc_gbak_missing_skipped_bytes                                = 336330952
	Isc_gbak_inv_skipped_bytes                                    = 336330953
	Isc_msgVerbose_write_charsets                                 = 336330963
	Isc_msgVerbose_write_collations                               = 336330964
	Isc_gbak_err_restore_charset                                  = 336330965
	Isc_msgVerbose_restore_charset                                = 336330966
	Isc_gbak_err_restore_collation                                = 336330967
	Isc_msgVerbose_restore_collation                              = 336330968
	Isc_gbak_read_error                                           = 336330972
	Isc_gbak_write_error                                          = 336330973
	Isc_gbak_db_in_use                                            = 336330985
	Isc_gbak_sysmemex                                             = 336330990
	Isc_write_role_1                                              = 336331000
	Isc_write_role_2                                              = 336331001
	Isc_gbak_restore_role_failed                                  = 336331002
	Isc_restore_role                                              = 336331003
	Isc_gbak_role_op                                              = 336331004
	Isc_gbak_role_op_missing                                      = 336331005
	Isc_gbak_convert_ext_tables                                   = 336331006
	Isc_gbak_warning                                              = 336331007
	Isc_gbak_error                                                = 336331008
	Isc_gbak_page_buffers                                         = 336331009
	Isc_gbak_page_buffers_missing                                 = 336331010
	Isc_gbak_page_buffers_wrong_param                             = 336331011
	Isc_gbak_page_buffers_restore                                 = 336331012
	Isc_gbak_inv_size                                             = 336331014
	Isc_gbak_file_outof_sequence                                  = 336331015
	Isc_gbak_join_file_missing                                    = 336331016
	Isc_gbak_stdin_not_supptd                                     = 336331017
	Isc_gbak_stdout_not_supptd                                    = 336331018
	Isc_gbak_bkup_corrupt                                         = 336331019
	Isc_gbak_unk_db_file_spec                                     = 336331020
	Isc_gbak_hdr_write_failed                                     = 336331021
	Isc_gbak_disk_space_ex                                        = 336331022
	Isc_gbak_size_lt_min                                          = 336331023
	Isc_gbak_svc_name_missing                                     = 336331025
	Isc_gbak_not_ownr                                             = 336331026
	Isc_gbak_opt_mode                                             = 336331030
	Isc_gbak_mode_req                                             = 336331031
	Isc_gbak_just_data                                            = 336331033
	Isc_gbak_data_only                                            = 336331034
	Isc_gbak_activating_idx                                       = 336331037
	Isc_write_map_1                                               = 336331048
	Isc_write_map_2                                               = 336331049
	Isc_get_map_1                                                 = 336331050
	Isc_get_map_2                                                 = 336331051
	Isc_get_map_3                                                 = 336331052
	Isc_get_map_4                                                 = 336331053
	Isc_gbak_missing_interval                                     = 336331078
	Isc_gbak_wrong_interval                                       = 336331079
	Isc_gbak_verify_verbint                                       = 336331081
	Isc_gbak_option_only_restore                                  = 336331082
	Isc_gbak_option_only_backup                                   = 336331083
	Isc_gbak_option_conflict                                      = 336331084
	Isc_gbak_param_conflict                                       = 336331085
	Isc_gbak_option_repeated                                      = 336331086
	Isc_gbak_max_dbkey_recursion                                  = 336331091
	Isc_gbak_max_dbkey_length                                     = 336331092
	Isc_gbak_invalid_metadata                                     = 336331093
	Isc_gbak_invalid_data                                         = 336331094
	Isc_gbak_inv_bkup_ver2                                        = 336331096
	Isc_gbak_db_format_too_old2                                   = 336331100
	Isc_gbak_missing_perf                                         = 336331118
	Isc_gbak_wrong_perf                                           = 336331119
	Isc_gbak_too_long_perf                                        = 336331120
	Isc_gbak_opt_replica                                          = 336331155
	Isc_gbak_replica_req                                          = 336331156
	Isc_gbak_missing_prl_wrks                                     = 336331159
	Isc_gbak_inv_prl_wrks                                         = 336331160
	Isc_gbak_plugin_schema_migration                              = 336331173
	Isc_gbak_plugin_schema_migration_err                          = 336331174
	Isc_gbak_writing_constants                                    = 336331175
	Isc_gbak_writing_constant                                     = 336331176
	Isc_gbak_constant                                             = 336331177
	Isc_gbak_restoring_constant                                   = 336331178
	Isc_gbak_db_not_online                                        = 336331762
	Isc_gbak_gfix_online                                          = 336331763
	Isc_gbak_db_not_online_reason                                 = 336331778
	Isc_dsql_too_old_ods                                          = 336397205
	Isc_dsql_table_not_found                                      = 336397206
	Isc_dsql_view_not_found                                       = 336397207
	Isc_dsql_line_col_error                                       = 336397208
	Isc_dsql_unknown_pos                                          = 336397209
	Isc_dsql_no_dup_name                                          = 336397210
	Isc_dsql_too_many_values                                      = 336397211
	Isc_dsql_no_array_computed                                    = 336397212
	Isc_dsql_implicit_domain_name                                 = 336397213
	Isc_dsql_only_can_subscript_array                             = 336397214
	Isc_dsql_max_sort_items                                       = 336397215
	Isc_dsql_max_group_items                                      = 336397216
	Isc_dsql_conflicting_sort_field                               = 336397217
	Isc_dsql_derived_table_more_columns                           = 336397218
	Isc_dsql_derived_table_less_columns                           = 336397219
	Isc_dsql_derived_field_unnamed                                = 336397220
	Isc_dsql_derived_field_dup_name                               = 336397221
	Isc_dsql_derived_alias_select                                 = 336397222
	Isc_dsql_derived_alias_field                                  = 336397223
	Isc_dsql_auto_field_bad_pos                                   = 336397224
	Isc_dsql_cte_wrong_reference                                  = 336397225
	Isc_dsql_cte_cycle                                            = 336397226
	Isc_dsql_cte_outer_join                                       = 336397227
	Isc_dsql_cte_mult_references                                  = 336397228
	Isc_dsql_cte_not_a_union                                      = 336397229
	Isc_dsql_cte_nonrecurs_after_recurs                           = 336397230
	Isc_dsql_cte_wrong_clause                                     = 336397231
	Isc_dsql_cte_union_all                                        = 336397232
	Isc_dsql_cte_miss_nonrecursive                                = 336397233
	Isc_dsql_cte_nested_with                                      = 336397234
	Isc_dsql_col_more_than_once_using                             = 336397235
	Isc_dsql_unsupp_feature_dialect                               = 336397236
	Isc_dsql_cte_not_used                                         = 336397237
	Isc_dsql_col_more_than_once_view                              = 336397238
	Isc_dsql_unsupported_in_auto_trans                            = 336397239
	Isc_dsql_eval_unknode                                         = 336397240
	Isc_dsql_agg_wrongarg                                         = 336397241
	Isc_dsql_agg2_wrongarg                                        = 336397242
	Isc_dsql_nodateortime_pm_string                               = 336397243
	Isc_dsql_invalid_datetime_subtract                            = 336397244
	Isc_dsql_invalid_dateortime_add                               = 336397245
	Isc_dsql_invalid_type_minus_date                              = 336397246
	Isc_dsql_nostring_addsub_dial3                                = 336397247
	Isc_dsql_invalid_type_addsub_dial3                            = 336397248
	Isc_dsql_invalid_type_multip_dial1                            = 336397249
	Isc_dsql_nostring_multip_dial3                                = 336397250
	Isc_dsql_invalid_type_multip_dial3                            = 336397251
	Isc_dsql_mustuse_numeric_div_dial1                            = 336397252
	Isc_dsql_nostring_div_dial3                                   = 336397253
	Isc_dsql_invalid_type_div_dial3                               = 336397254
	Isc_dsql_nostring_neg_dial3                                   = 336397255
	Isc_dsql_invalid_type_neg                                     = 336397256
	Isc_dsql_max_distinct_items                                   = 336397257
	Isc_dsql_alter_charset_failed                                 = 336397258
	Isc_dsql_comment_on_failed                                    = 336397259
	Isc_dsql_create_func_failed                                   = 336397260
	Isc_dsql_alter_func_failed                                    = 336397261
	Isc_dsql_create_alter_func_failed                             = 336397262
	Isc_dsql_drop_func_failed                                     = 336397263
	Isc_dsql_recreate_func_failed                                 = 336397264
	Isc_dsql_create_proc_failed                                   = 336397265
	Isc_dsql_alter_proc_failed                                    = 336397266
	Isc_dsql_create_alter_proc_failed                             = 336397267
	Isc_dsql_drop_proc_failed                                     = 336397268
	Isc_dsql_recreate_proc_failed                                 = 336397269
	Isc_dsql_create_trigger_failed                                = 336397270
	Isc_dsql_alter_trigger_failed                                 = 336397271
	Isc_dsql_create_alter_trigger_failed                          = 336397272
	Isc_dsql_drop_trigger_failed                                  = 336397273
	Isc_dsql_recreate_trigger_failed                              = 336397274
	Isc_dsql_create_collation_failed                              = 336397275
	Isc_dsql_drop_collation_failed                                = 336397276
	Isc_dsql_create_domain_failed                                 = 336397277
	Isc_dsql_alter_domain_failed                                  = 336397278
	Isc_dsql_drop_domain_failed                                   = 336397279
	Isc_dsql_create_except_failed                                 = 336397280
	Isc_dsql_alter_except_failed                                  = 336397281
	Isc_dsql_create_alter_except_failed                           = 336397282
	Isc_dsql_recreate_except_failed                               = 336397283
	Isc_dsql_drop_except_failed                                   = 336397284
	Isc_dsql_create_sequence_failed                               = 336397285
	Isc_dsql_create_table_failed                                  = 336397286
	Isc_dsql_alter_table_failed                                   = 336397287
	Isc_dsql_drop_table_failed                                    = 336397288
	Isc_dsql_recreate_table_failed                                = 336397289
	Isc_dsql_create_pack_failed                                   = 336397290
	Isc_dsql_alter_pack_failed                                    = 336397291
	Isc_dsql_create_alter_pack_failed                             = 336397292
	Isc_dsql_drop_pack_failed                                     = 336397293
	Isc_dsql_recreate_pack_failed                                 = 336397294
	Isc_dsql_create_pack_body_failed                              = 336397295
	Isc_dsql_drop_pack_body_failed                                = 336397296
	Isc_dsql_recreate_pack_body_failed                            = 336397297
	Isc_dsql_create_view_failed                                   = 336397298
	Isc_dsql_alter_view_failed                                    = 336397299
	Isc_dsql_create_alter_view_failed                             = 336397300
	Isc_dsql_recreate_view_failed                                 = 336397301
	Isc_dsql_drop_view_failed                                     = 336397302
	Isc_dsql_drop_sequence_failed                                 = 336397303
	Isc_dsql_recreate_sequence_failed                             = 336397304
	Isc_dsql_drop_index_failed                                    = 336397305
	Isc_dsql_drop_filter_failed                                   = 336397306
	Isc_dsql_drop_shadow_failed                                   = 336397307
	Isc_dsql_drop_role_failed                                     = 336397308
	Isc_dsql_drop_user_failed                                     = 336397309
	Isc_dsql_create_role_failed                                   = 336397310
	Isc_dsql_alter_role_failed                                    = 336397311
	Isc_dsql_alter_index_failed                                   = 336397312
	Isc_dsql_alter_database_failed                                = 336397313
	Isc_dsql_create_shadow_failed                                 = 336397314
	Isc_dsql_create_filter_failed                                 = 336397315
	Isc_dsql_create_index_failed                                  = 336397316
	Isc_dsql_create_user_failed                                   = 336397317
	Isc_dsql_alter_user_failed                                    = 336397318
	Isc_dsql_grant_failed                                         = 336397319
	Isc_dsql_revoke_failed                                        = 336397320
	Isc_dsql_cte_recursive_aggregate                              = 336397321
	Isc_dsql_mapping_failed                                       = 336397322
	Isc_dsql_alter_sequence_failed                                = 336397323
	Isc_dsql_create_generator_failed                              = 336397324
	Isc_dsql_set_generator_failed                                 = 336397325
	Isc_dsql_wlock_simple                                         = 336397326
	Isc_dsql_firstskip_rows                                       = 336397327
	Isc_dsql_wlock_aggregates                                     = 336397328
	Isc_dsql_wlock_conflict                                       = 336397329
	Isc_dsql_max_exception_arguments                              = 336397330
	Isc_dsql_string_byte_length                                   = 336397331
	Isc_dsql_string_char_length                                   = 336397332
	Isc_dsql_max_nesting                                          = 336397333
	Isc_dsql_recreate_user_failed                                 = 336397334
	Isc_dsql_table_value_many_columns                             = 336397335
	Isc_dsql_create_schema_failed                                 = 336397336
	Isc_dsql_drop_schema_failed                                   = 336397337
	Isc_dsql_recreate_schema_failed                               = 336397338
	Isc_dsql_alter_schema_failed                                  = 336397339
	Isc_dsql_create_alter_schema_failed                           = 336397340
	Isc_dsql_create_const_failed                                  = 336397341
	Isc_dsql_alter_const_failed                                   = 336397342
	Isc_dsql_create_alter_const_failed                            = 336397343
	Isc_dsql_create_policy_failed                                 = 336398288
	Isc_dsql_alter_policy_failed                                  = 336398289
	Isc_dsql_drop_policy_failed                                   = 336398290
	Isc_dsql_agg_wrongtype                                        = 336398291
	Isc_dsql_create_job_failed                                    = 336398292
	Isc_dsql_alter_job_failed                                     = 336398293
	Isc_dsql_create_alter_job_failed                              = 336398294
	Isc_dsql_drop_job_failed                                      = 336398295
	Isc_dsql_recreate_job_failed                                  = 336398296
	Isc_dsql_create_foreign_server_failed                         = 336398302
	Isc_dsql_alter_foreign_server_failed                          = 336398303
	Isc_dsql_create_alter_foreign_server_failed                   = 336398304
	Isc_dsql_recreate_foreign_server_failed                       = 336398305
	Isc_dsql_drop_foreign_server_failed                           = 336398306
	Isc_dsql_create_user_mapping_failed                           = 336398307
	Isc_dsql_alter_user_mapping_failed                            = 336398308
	Isc_dsql_create_alter_user_mapping_failed                     = 336398309
	Isc_dsql_recreate_user_mapping_failed                         = 336398310
	Isc_dsql_drop_user_mapping_failed                             = 336398311
	Isc_dsql_comment_on_user_mapping_failed                       = 336398312
	Isc_dsql_foreign_table_not_found                              = 336398313
	Isc_dsql_create_foreign_table_failed                          = 336398314
	Isc_dsql_alter_foreign_table_failed                           = 336398315
	Isc_dsql_drop_foreign_table_failed                            = 336398316
	Isc_dsql_recreate_foreign_table_failed                        = 336398317
	Isc_savepoint_error                                           = 336527650
	Isc_cache_non_zero_use_count                                  = 336527661
	Isc_unexpected_page_change                                    = 336527662
	Isc_rdb_triggers_rdb_flags_corrupt                            = 336527664
	Isc_GEN_ERR                                                   = 336658432
	Isc_USAGE                                                     = 336658433
	Isc_SWITCH                                                    = 336658434
	Isc_NO_DB                                                     = 336658435
	Isc_FILE_OPEN_ERR                                             = 336658436
	Isc_COMMIT_PROMPT                                             = 336658437
	Isc_COMMIT_MSG                                                = 336658438
	Isc_ROLLBACK_MSG                                              = 336658439
	Isc_CMD_ERR                                                   = 336658440
	Isc_ADD_PROMPT                                                = 336658441
	Isc_VERSION                                                   = 336658442
	Isc_USAGE_ALL                                                 = 336658443
	Isc_NUMBER_PAGES                                              = 336658444
	Isc_SWEEP_INTERV                                              = 336658445
	Isc_NUM_WAL_BUFF                                              = 336658446
	Isc_WAL_BUFF_SIZE                                             = 336658447
	Isc_CKPT_LENGTH                                               = 336658448
	Isc_CKPT_INTERV                                               = 336658449
	Isc_WAL_GRPC_WAIT                                             = 336658450
	Isc_BASE_LEVEL                                                = 336658451
	Isc_LIMBO                                                     = 336658452
	Isc_HLP_FRONTEND                                              = 336658453
	Isc_HLP_BLOBED                                                = 336658454
	Isc_HLP_BLOBDMP                                               = 336658455
	Isc_HLP_EDIT                                                  = 336658456
	Isc_HLP_INPUT                                                 = 336658457
	Isc_HLP_OUTPUT                                                = 336658458
	Isc_HLP_SHELL                                                 = 336658459
	Isc_HLP_HELP                                                  = 336658460
	Isc_HLP_SETCOM                                                = 336658461
	Isc_HLP_SET                                                   = 336658462
	Isc_HLP_SETAUTO                                               = 336658463
	Isc_HLP_SETBLOB                                               = 336658464
	Isc_HLP_SETCOUNT                                              = 336658465
	Isc_HLP_SETECHO                                               = 336658466
	Isc_HLP_SETSTAT                                               = 336658467
	Isc_HLP_SETTERM                                               = 336658468
	Isc_HLP_SHOW                                                  = 336658469
	Isc_HLP_OBJTYPE                                               = 336658470
	Isc_HLP_EXIT                                                  = 336658471
	Isc_HLP_QUIT                                                  = 336658472
	Isc_HLP_ALL                                                   = 336658473
	Isc_HLP_SETSCHEMA                                             = 336658474
	Isc_YES_ANS                                                   = 336658475
	Isc_REPORT1                                                   = 336658476
	Isc_REPORT2                                                   = 336658477
	Isc_BLOB_SUBTYPE                                              = 336658478
	Isc_BLOB_PROMPT                                               = 336658479
	Isc_DATE_PROMPT                                               = 336658480
	Isc_NAME_PROMPT                                               = 336658481
	Isc_DATE_ERR                                                  = 336658482
	Isc_CON_PROMPT                                                = 336658483
	Isc_HLP_SETLIST                                               = 336658484
	Isc_NOT_FOUND                                                 = 336658485
	Isc_COPY_ERR                                                  = 336658486
	Isc_SERVER_TOO_OLD                                            = 336658487
	Isc_REC_COUNT                                                 = 336658488
	Isc_UNLICENSED                                                = 336658489
	Isc_HLP_SETWIDTH                                              = 336658490
	Isc_HLP_SETPLAN                                               = 336658491
	Isc_HLP_SETTIME                                               = 336658492
	Isc_HLP_EDIT2                                                 = 336658493
	Isc_HLP_OUTPUT2                                               = 336658494
	Isc_HLP_SETNAMES                                              = 336658495
	Isc_HLP_OBJTYPE2                                              = 336658496
	Isc_HLP_SETBLOB2                                              = 336658497
	Isc_HLP_SET_ROOT                                              = 336658498
	Isc_NO_TABLES                                                 = 336658499
	Isc_NO_TABLE                                                  = 336658500
	Isc_NO_VIEWS                                                  = 336658501
	Isc_NO_VIEW                                                   = 336658502
	Isc_NO_INDICES_ON_REL                                         = 336658503
	Isc_NO_REL_OR_INDEX                                           = 336658504
	Isc_NO_INDICES                                                = 336658505
	Isc_NO_DOMAIN                                                 = 336658506
	Isc_NO_DOMAINS                                                = 336658507
	Isc_NO_EXCEPTION                                              = 336658508
	Isc_NO_EXCEPTIONS                                             = 336658509
	Isc_NO_FILTER                                                 = 336658510
	Isc_NO_FILTERS                                                = 336658511
	Isc_NO_FUNCTION                                               = 336658512
	Isc_NO_FUNCTIONS                                              = 336658513
	Isc_NO_GEN                                                    = 336658514
	Isc_NO_GENS                                                   = 336658515
	Isc_NO_GRANT_ON_REL                                           = 336658516
	Isc_NO_GRANT_ON_PROC                                          = 336658517
	Isc_NO_REL_OR_PROC                                            = 336658518
	Isc_NO_PROC                                                   = 336658519
	Isc_NO_PROCS                                                  = 336658520
	Isc_NO_TRIGGERS_ON_REL                                        = 336658521
	Isc_NO_REL_OR_TRIGGER                                         = 336658522
	Isc_NO_TRIGGERS                                               = 336658523
	Isc_NO_CHECKS_ON_REL                                          = 336658524
	Isc_REPORT2_WINDOWS_ONLY                                      = 336658525
	Isc_BUFFER_OVERFLOW                                           = 336658526
	Isc_NO_ROLES                                                  = 336658527
	Isc_NO_OBJECT                                                 = 336658528
	Isc_NO_GRANT_ON_ROL                                           = 336658529
	Isc_UNEXPECTED_EOF                                            = 336658530
	Isc_TIME_ERR                                                  = 336658533
	Isc_HLP_OBJTYPE3                                              = 336658534
	Isc_NO_ROLE                                                   = 336658535
	Isc_USAGE_BAIL                                                = 336658536
	Isc_HLP_SETSQLDIALECT                                         = 336658538
	Isc_NO_GRANT_ON_ANY                                           = 336658539
	Isc_HLP_SETPLANONLY                                           = 336658540
	Isc_HLP_SETHEADING                                            = 336658541
	Isc_HLP_SETBAIL                                               = 336658542
	Isc_USAGE_CACHE                                               = 336658543
	Isc_TIME_PROMPT                                               = 336658544
	Isc_TIMESTAMP_PROMPT                                          = 336658545
	Isc_TIMESTAMP_ERR                                             = 336658546
	Isc_NO_COMMENTS                                               = 336658547
	Isc_ONLY_FIRST_BLOBS                                          = 336658548
	Isc_MSG_TABLES                                                = 336658549
	Isc_MSG_FUNCTIONS                                             = 336658550
	Isc_EXACTLINE                                                 = 336658551
	Isc_AFTERLINE                                                 = 336658552
	Isc_NO_TRIGGER                                                = 336658553
	Isc_USAGE_CHARSET                                             = 336658554
	Isc_USAGE_DATABASE                                            = 336658555
	Isc_USAGE_ECHO                                                = 336658556
	Isc_USAGE_EXTRACT                                             = 336658557
	Isc_USAGE_INPUT                                               = 336658558
	Isc_USAGE_MERGE                                               = 336658559
	Isc_USAGE_MERGE2                                              = 336658560
	Isc_USAGE_NOAUTOCOMMIT                                        = 336658561
	Isc_USAGE_NOWARN                                              = 336658562
	Isc_USAGE_OUTPUT                                              = 336658563
	Isc_USAGE_PAGE                                                = 336658564
	Isc_USAGE_PASSWORD                                            = 336658565
	Isc_USAGE_QUIET                                               = 336658566
	Isc_USAGE_ROLE                                                = 336658567
	Isc_USAGE_ROLE2                                               = 336658568
	Isc_USAGE_SQLDIALECT                                          = 336658569
	Isc_USAGE_TERM                                                = 336658570
	Isc_USAGE_USER                                                = 336658571
	Isc_USAGE_XTRACT                                              = 336658572
	Isc_USAGE_VERSION                                             = 336658573
	Isc_USAGE_NOARG                                               = 336658574
	Isc_USAGE_NOTINT                                              = 336658575
	Isc_USAGE_RANGE                                               = 336658576
	Isc_USAGE_DUPSW                                               = 336658577
	Isc_USAGE_DUPDB                                               = 336658578
	Isc_NO_DEPENDENCIES                                           = 336658579
	Isc_NO_COLLATION                                              = 336658580
	Isc_NO_COLLATIONS                                             = 336658581
	Isc_MSG_COLLATIONS                                            = 336658582
	Isc_NO_SECCLASS                                               = 336658583
	Isc_NO_DB_WIDE_SECCLASS                                       = 336658584
	Isc_CANNOT_GET_SRV_VER                                        = 336658585
	Isc_USAGE_NODBTRIGGERS                                        = 336658586
	Isc_USAGE_TRUSTED                                             = 336658587
	Isc_BULK_PROMPT                                               = 336658588
	Isc_NO_CONNECTED_USERS                                        = 336658589
	Isc_USERS_IN_DB                                               = 336658590
	Isc_OUTPUT_TRUNCATED                                          = 336658591
	Isc_VALID_OPTIONS                                             = 336658592
	Isc_USAGE_FETCH                                               = 336658593
	Isc_PASS_FILE_OPEN                                            = 336658594
	Isc_PASS_FILE_READ                                            = 336658595
	Isc_EMPTY_PASS                                                = 336658596
	Isc_HLP_SETMAXROWS                                            = 336658597
	Isc_NO_PACKAGE                                                = 336658598
	Isc_NO_PACKAGES                                               = 336658599
	Isc_NO_SCHEMA                                                 = 336658600
	Isc_NO_SCHEMAS                                                = 336658601
	Isc_MAXROWS_INVALID                                           = 336658602
	Isc_MAXROWS_OUTOF_RANGE                                       = 336658603
	Isc_MAXROWS_NEGATIVE                                          = 336658604
	Isc_HLP_SETEXPLAIN                                            = 336658605
	Isc_NO_GRANT_ON_GEN                                           = 336658606
	Isc_NO_GRANT_ON_XCP                                           = 336658607
	Isc_NO_GRANT_ON_FLD                                           = 336658608
	Isc_NO_GRANT_ON_CS                                            = 336658609
	Isc_NO_GRANT_ON_COLL                                          = 336658610
	Isc_NO_GRANT_ON_PKG                                           = 336658611
	Isc_NO_GRANT_ON_FUN                                           = 336658612
	Isc_REPORT_NEW1                                               = 336658613
	Isc_REPORT_NEW2                                               = 336658614
	Isc_REPORT_NEW3                                               = 336658615
	Isc_NO_MAP                                                    = 336658616
	Isc_NO_MAPS                                                   = 336658617
	Isc_INVALID_TERM_CHARS                                        = 336658618
	Isc_REC_DISPLAYCOUNT                                          = 336658619
	Isc_COLUMNS_HIDDEN                                            = 336658620
	Isc_HLP_SETRECORDBUF                                          = 336658621
	Isc_NUMBER_USED_PAGES                                         = 336658622
	Isc_NUMBER_FREE_PAGES                                         = 336658623
	Isc_DATABASE_CRYPTED                                          = 336658624
	Isc_DATABASE_NOT_CRYPTED                                      = 336658625
	Isc_DATABASE_CRYPT_PROCESS                                    = 336658626
	Isc_MSG_ROLES                                                 = 336658627
	Isc_NO_TIMEOUTS                                               = 336658628
	Isc_HLP_SETKEEPTRAN                                           = 336658629
	Isc_HLP_SETPERTAB                                             = 336658630
	Isc_BAD_STMT_TYPE                                             = 336658631
	Isc_MSG_PACKAGES                                              = 336658632
	Isc_NO_PUBLICATION                                            = 336658633
	Isc_NO_PUBLICATIONS                                           = 336658634
	Isc_MSG_PUBLICATIONS                                          = 336658635
	Isc_MSG_PROCEDURES                                            = 336658636
	Isc_HLP_EXPLAIN                                               = 336658637
	Isc_USAGE_AUTOTERM                                            = 336658638
	Isc_AUTOTERM_NOT_SUPPORTED                                    = 336658639
	Isc_HLP_SETAUTOTERM                                           = 336658640
	Isc_HLP_SETWIRESTATS                                          = 336658641
	Isc_USAGE_SEARCH_PATH                                         = 336658642
	Isc_MSG_SCHEMAS                                               = 336658643
	Isc_NO_CONSTANTS                                              = 336658644
	Isc_NO_CONSTANT                                               = 336658645
	Isc_NO_OBJECT_BY_PATTERN                                      = 336659432
	Isc_MSG_GENERATORS                                            = 336659433
	Isc_MSG_VIEWS                                                 = 336659434
	Isc_MSG_INDICIES                                              = 336659435
	Isc_MSG_CONSTRAINS                                            = 336659436
	Isc_MSG_DOMAINS                                               = 336659437
	Isc_MSG_TRIGGERS                                              = 336659438
	Isc_MSG_EXCEPTIONS                                            = 336659441
	Isc_MSG_GRANTS                                                = 336659442
	Isc_MSG_COMMENTS                                              = 336659443
	Isc_MSG_FILTERS                                               = 336659444
	Isc_MSG_DEPENDENCIES                                          = 336659445
	Isc_MSG_SECCLASSES                                            = 336659446
	Isc_MSG_SYSTEM                                                = 336659447
	Isc_MSG_MAP                                                   = 336659449
	Isc_USAGE_CERTIFICATE                                         = 336659450
	Isc_USAGE_PIN                                                 = 336659451
	Isc_USAGE_VERIFY                                              = 336659452
	Isc_USAGE_LOGIN                                               = 336659453
	Isc_CLEAR_HISTORY                                             = 336659454
	Isc_MSG_SYS_FUNCTIONS                                         = 336659459
	Isc_MSG_SYS_PROCEDURES                                        = 336659460
	Isc_MSG_SYS_GENERATORS                                        = 336659461
	Isc_MSG_SYS_COLLATIONS                                        = 336659462
	Isc_MSG_SYS_PACKAGES                                          = 336659463
	Isc_MSG_SYS_PUBLICATIONS                                      = 336659464
	Isc_MSG_SYS_TABLES                                            = 336659465
	Isc_MSG_JOBS                                                  = 336659466
	Isc_NO_JOB                                                    = 336659467
	Isc_NO_JOBS                                                   = 336659468
	Isc_NO_SYSTEM_COLLATIONS                                      = 336659469
	Isc_NO_SYSTEM_FUNCTIONS                                       = 336659470
	Isc_NO_SYSTEM_TABLES                                          = 336659471
	Isc_NO_SYSTEM_GENS                                            = 336659472
	Isc_NO_SYSTEM_PROCS                                           = 336659473
	Isc_NO_SYSTEM_PACKAGES                                        = 336659474
	Isc_NO_SYSTEM_PUBLICATIONS                                    = 336659475
	Isc_NO_SYSTEM_COLLATION                                       = 336659476
	Isc_NO_SYSTEM_FUNCTION                                        = 336659477
	Isc_NO_SYSTEM_TABLE                                           = 336659478
	Isc_NO_SYSTEM_GEN                                             = 336659479
	Isc_NO_SYSTEM_PROC                                            = 336659480
	Isc_NO_SYSTEM_PACKAGE                                         = 336659481
	Isc_NO_SYSTEM_PUBLICATION                                     = 336659482
	Isc_NO_ANY_DEPENDENCIES                                       = 336659483
	Isc_MSG_SYS_ROLES                                             = 336659484
	Isc_NO_SYSTEM_ROLE                                            = 336659485
	Isc_USAGE_EXTRACT_NO_TERM                                     = 336659486
	Isc_USAGE_TOKEN                                               = 336659487
	Isc_MSG_FOREIGN_SERVERS                                       = 336659488
	Isc_MSG_FOREIGN_USER_MAPPINGS                                 = 336659489
	Isc_NO_FOREIGN_SERVER                                         = 336659490
	Isc_NO_FOREIGN_SERVERS                                        = 336659491
	Isc_NO_FOREIGN_USER_MAPPING                                   = 336659492
	Isc_NO_FOREIGN_USER_MAPPINGS                                  = 336659493
	Isc_USAGE_READ                                                = 336659494
	Isc_GsecMsg1                                                  = 336723969
	Isc_GsecMsg2                                                  = 336723970
	Isc_GsecMsg3                                                  = 336723971
	Isc_GsecMsg4                                                  = 336723972
	Isc_GsecMsg5                                                  = 336723973
	Isc_GsecMsg6                                                  = 336723974
	Isc_GsecMsg7                                                  = 336723975
	Isc_GsecMsg8                                                  = 336723976
	Isc_GsecMsg9                                                  = 336723977
	Isc_GsecMsg10                                                 = 336723978
	Isc_GsecMsg11                                                 = 336723979
	Isc_GsecMsg12                                                 = 336723980
	Isc_GsecMsg13                                                 = 336723981
	Isc_GsecMsg14                                                 = 336723982
	Isc_gsec_cant_open_db                                         = 336723983
	Isc_gsec_switches_error                                       = 336723984
	Isc_gsec_no_op_spec                                           = 336723985
	Isc_gsec_no_usr_name                                          = 336723986
	Isc_gsec_err_add                                              = 336723987
	Isc_gsec_err_modify                                           = 336723988
	Isc_gsec_err_find_mod                                         = 336723989
	Isc_gsec_err_rec_not_found                                    = 336723990
	Isc_gsec_err_delete                                           = 336723991
	Isc_gsec_err_find_del                                         = 336723992
	Isc_GsecMsg25                                                 = 336723993
	Isc_GsecMsg26                                                 = 336723994
	Isc_GsecMsg27                                                 = 336723995
	Isc_gsec_err_find_disp                                        = 336723996
	Isc_gsec_inv_param                                            = 336723997
	Isc_gsec_op_specified                                         = 336723998
	Isc_gsec_pw_specified                                         = 336723999
	Isc_gsec_uid_specified                                        = 336724000
	Isc_gsec_gid_specified                                        = 336724001
	Isc_gsec_proj_specified                                       = 336724002
	Isc_gsec_org_specified                                        = 336724003
	Isc_gsec_fname_specified                                      = 336724004
	Isc_gsec_mname_specified                                      = 336724005
	Isc_gsec_lname_specified                                      = 336724006
	Isc_GsecMsg39                                                 = 336724007
	Isc_gsec_inv_switch                                           = 336724008
	Isc_gsec_amb_switch                                           = 336724009
	Isc_gsec_no_op_specified                                      = 336724010
	Isc_gsec_params_not_allowed                                   = 336724011
	Isc_gsec_incompat_switch                                      = 336724012
	Isc_GsecMsg45                                                 = 336724013
	Isc_GsecMsg46                                                 = 336724014
	Isc_GsecMsg47                                                 = 336724015
	Isc_GsecMsg48                                                 = 336724016
	Isc_GsecMsg49                                                 = 336724017
	Isc_GsecMsgs50                                                = 336724018
	Isc_GsecMsg51                                                 = 336724019
	Isc_GsecMsg52                                                 = 336724020
	Isc_GsecMsg53                                                 = 336724021
	Isc_GsecMsg54                                                 = 336724022
	Isc_GsecMsg55                                                 = 336724023
	Isc_GsecMsg56                                                 = 336724024
	Isc_GsecMsg57                                                 = 336724025
	Isc_GsecMsg58                                                 = 336724026
	Isc_GsecMsg59                                                 = 336724027
	Isc_GsecMsg60                                                 = 336724028
	Isc_GsecMsg61                                                 = 336724029
	Isc_GsecMsg62                                                 = 336724030
	Isc_GsecMsg63                                                 = 336724031
	Isc_GsecMsg64                                                 = 336724032
	Isc_GsecMsg65                                                 = 336724033
	Isc_GsecMsg66                                                 = 336724034
	Isc_GsecMsg67                                                 = 336724035
	Isc_GsecMsg68                                                 = 336724036
	Isc_GsecMsg69                                                 = 336724037
	Isc_GsecMsg70                                                 = 336724038
	Isc_GsecMsg71                                                 = 336724039
	Isc_GsecMsg72                                                 = 336724040
	Isc_GsecMsg73                                                 = 336724041
	Isc_gsec_inv_username                                         = 336724044
	Isc_gsec_inv_pw_length                                        = 336724045
	Isc_gsec_db_specified                                         = 336724046
	Isc_gsec_db_admin_specified                                   = 336724047
	Isc_gsec_db_admin_pw_specified                                = 336724048
	Isc_gsec_sql_role_specified                                   = 336724049
	Isc_GsecMsg82                                                 = 336724050
	Isc_GsecMsg83                                                 = 336724051
	Isc_GsecMsg84                                                 = 336724052
	Isc_GsecMsg85                                                 = 336724053
	Isc_GsecMsg86                                                 = 336724054
	Isc_GsecMsg87                                                 = 336724055
	Isc_GsecMsg88                                                 = 336724056
	Isc_GsecMsg89                                                 = 336724057
	Isc_GsecMsg90                                                 = 336724058
	Isc_GsecMsg91                                                 = 336724059
	Isc_GsecMsg92                                                 = 336724060
	Isc_GsecMsg93                                                 = 336724061
	Isc_GsecMsg94                                                 = 336724062
	Isc_GsecMsg95                                                 = 336724063
	Isc_GsecMsg96                                                 = 336724064
	Isc_GsecMsg97                                                 = 336724065
	Isc_GsecMsg98                                                 = 336724066
	Isc_GsecMsg99                                                 = 336724067
	Isc_GsecMsg100                                                = 336724068
	Isc_GsecMsg101                                                = 336724069
	Isc_GsecMsg102                                                = 336724070
	Isc_GsecMsg103                                                = 336724071
	Isc_GsecMsg104                                                = 336724072
	Isc_GsecMsg200                                                = 336724168
	Isc_GsecMsg201                                                = 336724169
	Isc_GsecMsg202                                                = 336724170
	Isc_gstat_unknown_switch                                      = 336920577
	Isc_gstat_retry                                               = 336920578
	Isc_gstat_wrong_ods                                           = 336920579
	Isc_gstat_unexpected_eof                                      = 336920580
	Isc_gstat_open_err                                            = 336920605
	Isc_gstat_read_err                                            = 336920606
	Isc_gstat_sysmemex                                            = 336920607
	Isc_gstat_username                                            = 336920608
	Isc_gstat_password                                            = 336920609
	Isc_fbsvcmgr_bad_am                                           = 336986113
	Isc_fbsvcmgr_bad_wm                                           = 336986114
	Isc_fbsvcmgr_bad_rs                                           = 336986115
	Isc_fbsvcmgr_info_err                                         = 336986116
	Isc_fbsvcmgr_query_err                                        = 336986117
	Isc_fbsvcmgr_switch_unknown                                   = 336986118
	Isc_fbsvcmgr_bad_sm                                           = 336986159
	Isc_fbsvcmgr_fp_open                                          = 336986160
	Isc_fbsvcmgr_fp_read                                          = 336986161
	Isc_fbsvcmgr_fp_empty                                         = 336986162
	Isc_fbsvcmgr_bad_arg                                          = 336986164
	Isc_fbsvcmgr_info_limbo                                       = 336986170
	Isc_fbsvcmgr_limbo_state                                      = 336986171
	Isc_fbsvcmgr_limbo_advise                                     = 336986172
	Isc_fbsvcmgr_bad_rm                                           = 336986173
	Isc_utl_trusted_switch                                        = 337051649
	Isc_nbackup_missing_param                                     = 337117213
	Isc_nbackup_allowed_switches                                  = 337117214
	Isc_nbackup_unknown_param                                     = 337117215
	Isc_nbackup_unknown_switch                                    = 337117216
	Isc_nbackup_nofetchpw_svc                                     = 337117217
	Isc_nbackup_pwfile_error                                      = 337117218
	Isc_nbackup_size_with_lock                                    = 337117219
	Isc_nbackup_no_switch                                         = 337117220
	Isc_nbackup_err_read                                          = 337117223
	Isc_nbackup_err_write                                         = 337117224
	Isc_nbackup_err_seek                                          = 337117225
	Isc_nbackup_err_opendb                                        = 337117226
	Isc_nbackup_err_fadvice                                       = 337117227
	Isc_nbackup_err_createdb                                      = 337117228
	Isc_nbackup_err_openbk                                        = 337117229
	Isc_nbackup_err_createbk                                      = 337117230
	Isc_nbackup_err_eofdb                                         = 337117231
	Isc_nbackup_fixup_wrongstate                                  = 337117232
	Isc_nbackup_err_db                                            = 337117233
	Isc_nbackup_userpw_toolong                                    = 337117234
	Isc_nbackup_lostrec_db                                        = 337117235
	Isc_nbackup_lostguid_db                                       = 337117236
	Isc_nbackup_err_eofhdrdb                                      = 337117237
	Isc_nbackup_db_notlock                                        = 337117238
	Isc_nbackup_lostguid_bk                                       = 337117239
	Isc_nbackup_page_changed                                      = 337117240
	Isc_nbackup_dbsize_inconsistent                               = 337117241
	Isc_nbackup_failed_lzbk                                       = 337117242
	Isc_nbackup_err_eofhdrbk                                      = 337117243
	Isc_nbackup_invalid_incbk                                     = 337117244
	Isc_nbackup_unsupvers_incbk                                   = 337117245
	Isc_nbackup_invlevel_incbk                                    = 337117246
	Isc_nbackup_wrong_orderbk                                     = 337117247
	Isc_nbackup_err_eofbk                                         = 337117248
	Isc_nbackup_err_copy                                          = 337117249
	Isc_nbackup_err_eofhdr_restdb                                 = 337117250
	Isc_nbackup_lostguid_l0bk                                     = 337117251
	Isc_nbackup_switchd_parameter                                 = 337117255
	Isc_nbackup_user_stop                                         = 337117257
	Isc_nbackup_deco_parse                                        = 337117259
	Isc_nbackup_lostrec_guid_db                                   = 337117261
	Isc_nbackup_seq_misuse                                        = 337117265
	Isc_nbackup_wrong_param                                       = 337117268
	Isc_nbackup_clean_hist_misuse                                 = 337117269
	Isc_nbackup_clean_hist_missed                                 = 337117270
	Isc_nbackup_keep_hist_missed                                  = 337117271
	Isc_nbackup_second_keep_switch                                = 337117272
	Isc_trace_conflict_acts                                       = 337182750
	Isc_trace_act_notfound                                        = 337182751
	Isc_trace_switch_once                                         = 337182752
	Isc_trace_param_val_miss                                      = 337182753
	Isc_trace_param_invalid                                       = 337182754
	Isc_trace_switch_unknown                                      = 337182755
	Isc_trace_switch_svc_only                                     = 337182756
	Isc_trace_switch_user_only                                    = 337182757
	Isc_trace_switch_param_miss                                   = 337182758
	Isc_trace_param_act_notcompat                                 = 337182759
	Isc_trace_mandatory_switch_miss                               = 337182760
	Isc_diff_incmp_sw                                             = 337510415
	Isc_diff_unicode_init_error                                   = 337510416
	Isc_diff_replconf_section_not_found                           = 337510417
	Isc_DiffErr_slave_name                                        = 337510418
	Isc_DiffErr_records_number                                    = 337510419
	Isc_DiffErr_records                                           = 337510420
	Isc_DiffErr_tables_number                                     = 337510421
	Isc_DiffErr_tables_list                                       = 337510422
	Isc_DiffErr_primary_keys                                      = 337510423
	Isc_DiffErr_blobs                                             = 337510424
	Isc_DiffErr_remote_master                                     = 337510425
	Isc_DiffErr_sync_timeout                                      = 337510426
	Isc_DiffMgs_waiting                                           = 337510427
	Isc_DiffMgs_checking                                          = 337510428
	Isc_DiffMgs_failed                                            = 337510429
	Isc_DiffMgs_ok_empty                                          = 337510430
	Isc_DiffMgs_on_compared                                       = 337510431
	Isc_DiffMgs_summary_no_diff                                   = 337510432
	Isc_DiffMgs_summary_diff                                      = 337510433
	Isc_DiffMgs_summary_error                                     = 337510434
	Isc_DiffMgs_succeeded                                         = 337510435
	Isc_DiffMsg_version                                           = 337510436
	Isc_diff_param_val_miss                                       = 337510437
	Isc_DiffMsg_table_name                                        = 337510439
	Isc_DiffErr_index_name                                        = 337510440
	Isc_DiffErr_attributes                                        = 337510441
	Isc_DiffErr_field_count                                       = 337510442
	Isc_DiffErr_field_names                                       = 337510443
	Isc_DiffErrFull_not_found                                     = 337510444
	Isc_DiffErrFull_attributes                                    = 337510445
	Isc_DiffErrFull_blobs                                         = 337510446
	Isc_DiffErrFull_records                                       = 337510447
	Isc_DiffErrFull_master_has_more_records                       = 337510448
	Isc_DiffErrFull_slave_has_more_records                        = 337510449
	Isc_DiffErrFull_table_not_found                               = 337510450
	Isc_DiffErrFull_master_has_more_tables                        = 337510451
	Isc_DiffErrFull_slave_has_more_tables                         = 337510452
	Isc_DiffErrFull_records_value                                 = 337510453
)

var errmsgs = map[int]fberror{
	335544320:                             {"", 0, ""},
	Isc_arith_except:                      {"arithmetic exception, numeric overflow, or string truncation", 0, "22000"},
	Isc_bad_dbkey:                         {"invalid database key", 0, "HY000"},
	Isc_bad_db_format:                     {"file @1 is not a valid database", 1, "HY000"},
	Isc_bad_db_handle:                     {"invalid database handle (no active connection)", 0, "08003"},
	Isc_bad_dpb_content:                   {"bad parameters on attach or create database", 0, "HY000"},
	Isc_bad_dpb_form:                      {"unrecognized database parameter block", 0, "HY000"},
	Isc_bad_req_handle:                    {"invalid request handle", 0, "42000"},
	Isc_bad_segstr_handle:                 {"invalid BLOB handle", 0, "42000"},
	Isc_bad_segstr_id:                     {"invalid BLOB ID", 0, "42000"},
	Isc_bad_tpb_content:                   {"invalid parameter in transaction parameter block", 0, "HY000"},
	Isc_bad_tpb_form:                      {"invalid format for transaction parameter block", 0, "HY000"},
	Isc_bad_trans_handle:                  {"invalid transaction handle (expecting explicit transaction start)", 0, "HY000"},
	Isc_bug_check:                         {"internal Red Database consistency check (@1)", 1, "XX000"},
	Isc_convert_error:                     {"conversion error from string \"@1\"", 1, "22018"},
	Isc_db_corrupt:                        {"database file appears corrupt (@1)", 1, "XX001"},
	Isc_deadlock:                          {"deadlock", 0, "40001"},
	Isc_excess_trans:                      {"attempt to start more than @1 transactions", 1, "HY000"},
	Isc_from_no_match:                     {"no match for first value expression", 0, "01000"},
	Isc_infinap:                           {"information type inappropriate for object specified", 0, "HY000"},
	Isc_infona:                            {"no information of this type available for object specified", 0, "HY000"},
	Isc_infunk:                            {"unknown information item", 0, "HY000"},
	Isc_integ_fail:                        {"action cancelled by trigger (@1) to preserve data integrity", 1, "27000"},
	Isc_invalid_blr:                       {"invalid request BLR at offset @1", 1, "HY000"},
	Isc_io_error:                          {"I/O error during \"@1\" operation for file \"@2\"", 2, "08001"},
	Isc_lock_conflict:                     {"lock conflict on no wait transaction", 0, "40001"},
	Isc_metadata_corrupt:                  {"corrupt system table", 0, "XX001"},
	Isc_not_valid:                         {"validation error for column @1, value \"@2\"", 2, "23000"},
	Isc_no_cur_rec:                        {"no current record for fetch operation", 0, "22000"},
	Isc_no_dup:                            {"attempt to store duplicate value (visible to active transactions) in unique index @1", 1, "23000"},
	Isc_no_finish:                         {"program attempted to exit without finishing database", 0, "HY000"},
	Isc_no_meta_update:                    {"unsuccessful metadata update", 0, "42000"},
	Isc_no_priv:                           {"no permission for @1 access to @2 @3", 3, "28000"},
	Isc_no_recon:                          {"transaction is not in limbo", 0, "HY000"},
	Isc_no_record:                         {"invalid database key", 0, "01000"},
	Isc_no_segstr_close:                   {"BLOB was not closed", 0, "HY000"},
	Isc_obsolete_metadata:                 {"metadata is obsolete", 0, "HY000"},
	Isc_open_trans:                        {"cannot disconnect database with open transactions (@1 active)", 1, "01002"},
	Isc_port_len:                          {"message length error (encountered @1, expected @2)", 2, "HY000"},
	Isc_read_only_field:                   {"attempted update of read-only column @1", 1, "42000"},
	Isc_read_only_rel:                     {"attempted update of read-only table", 0, "42000"},
	Isc_read_only_trans:                   {"attempted update during read-only transaction", 0, "25006"},
	Isc_read_only_view:                    {"cannot update read-only view @1", 1, "42000"},
	Isc_req_no_trans:                      {"no transaction for request", 0, "25000"},
	Isc_req_sync:                          {"request synchronization error", 0, "HY000"},
	Isc_req_wrong_db:                      {"request referenced an unavailable database", 0, "HY000"},
	Isc_segment:                           {"segment buffer length shorter than expected", 0, "01000"},
	Isc_segstr_eof:                        {"attempted retrieval of more segments than exist", 0, "01000"},
	Isc_segstr_no_op:                      {"attempted invalid operation on a BLOB", 0, "HY000"},
	Isc_segstr_no_read:                    {"attempted read of a new, open BLOB", 0, "HY000"},
	Isc_segstr_no_trans:                   {"attempted action on BLOB outside transaction", 0, "HY000"},
	Isc_segstr_no_write:                   {"attempted write to read-only BLOB", 0, "HY000"},
	Isc_segstr_wrong_db:                   {"attempted reference to BLOB in unavailable database", 0, "HY000"},
	Isc_sys_request:                       {"operating system directive @1 failed", 1, "HY000"},
	Isc_stream_eof:                        {"attempt to fetch past the last record in a record stream", 0, "HY109"},
	Isc_unavailable:                       {"unavailable database", 0, "08001"},
	Isc_unres_rel:                         {"table @1 was omitted from the transaction reserving list", 1, "HY000"},
	Isc_uns_ext:                           {"request includes a DSRI extension not supported in this implementation", 0, "HY000"},
	Isc_wish_list:                         {"feature is not supported", 0, "0A000"},
	Isc_wrong_ods:                         {"unsupported on-disk structure for file @1; found @2.@3, support @4.@5", 5, "HY000"},
	Isc_wronumarg:                         {"wrong number of arguments on call", 0, "07002"},
	Isc_imp_exc:                           {"Implementation limit exceeded", 0, "54000"},
	Isc_random:                            {"@1", 1, "HY000"},
	Isc_fatal_conflict:                    {"unrecoverable conflict with limbo transaction @1", 1, "HY000"},
	Isc_badblk:                            {"internal error", 0, "XX000"},
	Isc_invpoolcl:                         {"internal error", 0, "XX000"},
	Isc_nopoolids:                         {"too many requests", 0, "HY000"},
	Isc_relbadblk:                         {"internal error", 0, "XX000"},
	Isc_blktoobig:                         {"block size exceeds implementation restriction", 0, "54000"},
	Isc_bufexh:                            {"buffer exhausted", 0, "HY001"},
	Isc_syntaxerr:                         {"BLR syntax error: expected @1 at offset @2, encountered @3", 3, "HY000"},
	Isc_bufinuse:                          {"buffer in use", 0, "HY000"},
	Isc_bdbincon:                          {"internal error", 0, "XX000"},
	Isc_reqinuse:                          {"request in use", 0, "HY000"},
	Isc_badodsver:                         {"incompatible version of on-disk structure", 0, "HY000"},
	Isc_relnotdef:                         {"table @1 is not defined", 1, "42S02"},
	Isc_fldnotdef:                         {"column @1 is not defined in table @2", 2, "42S22"},
	Isc_dirtypage:                         {"internal error", 0, "XX000"},
	Isc_waifortra:                         {"internal error", 0, "XX000"},
	Isc_doubleloc:                         {"internal error", 0, "XX000"},
	Isc_nodnotfnd:                         {"internal error", 0, "XX000"},
	Isc_dupnodfnd:                         {"internal error", 0, "XX000"},
	Isc_locnotmar:                         {"internal error", 0, "XX000"},
	Isc_badpagtyp:                         {"page @1 is of wrong type (expected @2, found @3)", 3, "42000"},
	Isc_corrupt:                           {"database corrupted", 0, "XX000"},
	Isc_badpage:                           {"checksum error on database page @1", 1, "42000"},
	Isc_badindex:                          {"index is broken", 0, "XX002"},
	Isc_dbbnotzer:                         {"database handle not zero", 0, "HY000"},
	Isc_tranotzer:                         {"transaction handle not zero", 0, "25000"},
	Isc_trareqmis:                         {"transaction--request mismatch (synchronization error)", 0, "25000"},
	Isc_badhndcnt:                         {"bad handle count", 0, "HY000"},
	Isc_wrotpbver:                         {"wrong version of transaction parameter block", 0, "HY000"},
	Isc_wroblrver:                         {"unsupported BLR version (expected @1, encountered @2)", 2, "HY000"},
	Isc_wrodpbver:                         {"wrong version of database parameter block", 0, "HY000"},
	Isc_blobnotsup:                        {"BLOB and array data types are not supported for @1 operation", 1, "42000"},
	Isc_badrelation:                       {"database corrupted", 0, "XX001"},
	Isc_nodetach:                          {"internal error", 0, "XX000"},
	Isc_notremote:                         {"internal error", 0, "XX000"},
	Isc_trainlim:                          {"transaction in limbo", 0, "25000"},
	Isc_notinlim:                          {"transaction not in limbo", 0, "25000"},
	Isc_traoutsta:                         {"transaction outstanding", 0, "25000"},
	Isc_connect_reject:                    {"connection rejected by remote interface", 0, "08004"},
	Isc_dbfile:                            {"internal error", 0, "XX000"},
	Isc_orphan:                            {"internal error", 0, "XX000"},
	Isc_no_lock_mgr:                       {"no lock manager available", 0, "HY000"},
	Isc_ctxinuse:                          {"context already in use (BLR error)", 0, "HY000"},
	Isc_ctxnotdef:                         {"context not defined (BLR error)", 0, "HY000"},
	Isc_datnotsup:                         {"data operation not supported", 0, "42000"},
	Isc_badmsgnum:                         {"undefined message number", 0, "HY000"},
	Isc_badparnum:                         {"undefined parameter number", 0, "HY000"},
	Isc_virmemexh:                         {"unable to allocate memory from operating system", 0, "HY001"},
	Isc_blocking_signal:                   {"blocking signal has been received", 0, "HY000"},
	Isc_lockmanerr:                        {"lock manager error", 0, "40001"},
	Isc_journerr:                          {"communication error with journal \"@1\"", 1, "HY000"},
	Isc_keytoobig:                         {"key size exceeds implementation restriction for index @1", 1, "42000"},
	Isc_nullsegkey:                        {"null segment of UNIQUE KEY", 0, "HY000"},
	Isc_sqlerr:                            {"SQL error code = @1", 1, "00000"},
	Isc_wrodynver:                         {"wrong DYN version", 0, "HY000"},
	Isc_funnotdef:                         {"function @1 is not defined", 1, "39000"},
	Isc_funmismat:                         {"function @1 could not be matched", 1, "39000"},
	Isc_bad_msg_vec:                       {"", 0, "22001"},
	Isc_bad_detach:                        {"database detach completed with errors", 0, "08007"},
	Isc_noargacc_read:                     {"database system cannot read argument @1", 1, "HY000"},
	Isc_noargacc_write:                    {"database system cannot write argument @1", 1, "HY000"},
	Isc_read_only:                         {"operation not supported", 0, "42000"},
	Isc_ext_err:                           {"@1 extension error", 1, "42000"},
	Isc_non_updatable:                     {"not updatable", 0, "42000"},
	Isc_no_rollback:                       {"no rollback performed", 0, "25S02"},
	Isc_bad_sec_info:                      {"", 0, "HY000"},
	Isc_invalid_sec_info:                  {"", 0, "28000"},
	Isc_misc_interpreted:                  {"@1", 1, "HY000"},
	Isc_update_conflict:                   {"update conflicts with concurrent update", 0, "40001"},
	Isc_unlicensed:                        {"product @1 is not licensed", 1, "HY000"},
	Isc_obj_in_use:                        {"object @1 is in use", 1, "42000"},
	Isc_nofilter:                          {"filter not found to convert type @1 to type @2", 2, "HY000"},
	Isc_shadow_accessed:                   {"cannot attach active shadow file", 0, "00000"},
	Isc_invalid_sdl:                       {"invalid slice description language at offset @1", 1, "HY000"},
	Isc_out_of_bounds:                     {"subscript out of bounds", 0, "42000"},
	Isc_invalid_dimension:                 {"column not array or invalid dimensions (expected @1, encountered @2)", 2, "42000"},
	Isc_rec_in_limbo:                      {"record from transaction @1 is stuck in limbo", 1, "HY000"},
	Isc_shadow_missing:                    {"a file in manual shadow @1 is unavailable", 1, "HY000"},
	Isc_cant_validate:                     {"secondary server attachments cannot validate databases", 0, "HY000"},
	Isc_cant_start_journal:                {"secondary server attachments cannot start journaling", 0, "HY000"},
	Isc_gennotdef:                         {"generator @1 is not defined", 1, "42000"},
	Isc_cant_start_logging:                {"secondary server attachments cannot start logging", 0, "HY000"},
	Isc_bad_segstr_type:                   {"invalid BLOB type for operation", 0, "42000"},
	Isc_foreign_key:                       {"violation of FOREIGN KEY constraint @1 on table @2", 2, "23000"},
	Isc_high_minor:                        {"minor version too high found @1 expected @2", 2, "HY000"},
	Isc_tra_state:                         {"transaction @1 is @2", 2, "00000"},
	Isc_trans_invalid:                     {"transaction marked invalid and cannot be committed", 0, "25000"},
	Isc_buf_invalid:                       {"cache buffer for page @1 invalid", 1, "HY000"},
	Isc_indexnotdefined:                   {"there is no index in table @1 with id @2", 2, "42S12"},
	Isc_login:                             {"Your user name and password are not defined. Ask your database administrator to set up a Red Database login.", 0, "28000"},
	Isc_invalid_bookmark:                  {"invalid bookmark handle", 0, "HY111"},
	Isc_bad_lock_level:                    {"invalid lock level @1", 1, "HY000"},
	Isc_relation_lock:                     {"lock on table @1 conflicts with existing lock", 1, "40001"},
	Isc_record_lock:                       {"requested record lock conflicts with existing lock", 0, "40001"},
	Isc_max_idx:                           {"maximum indexes per table (@1) exceeded", 1, "54000"},
	Isc_jrn_enable:                        {"enable journal for database before starting online dump", 0, "HY000"},
	Isc_old_failure:                       {"online dump failure. Retry dump", 0, "HY000"},
	Isc_old_in_progress:                   {"an online dump is already in progress", 0, "HY000"},
	Isc_old_no_space:                      {"no more disk/tape space.  Cannot continue online dump", 0, "HY000"},
	Isc_no_wal_no_jrn:                     {"journaling allowed only if database has Write-ahead Log", 0, "HY000"},
	Isc_num_old_files:                     {"maximum number of online dump files that can be specified is 16", 0, "HY000"},
	Isc_wal_file_open:                     {"error in opening Write-ahead Log file during recovery", 0, "HY000"},
	Isc_bad_stmt_handle:                   {"invalid statement handle", 0, "26000"},
	Isc_wal_failure:                       {"Write-ahead log subsystem failure", 0, "HY000"},
	Isc_walw_err:                          {"WAL Writer error", 0, "HY000"},
	Isc_logh_small:                        {"Log file header of @1 too small", 1, "HY000"},
	Isc_logh_inv_version:                  {"Invalid version of log file @1", 1, "HY000"},
	Isc_logh_open_flag:                    {"Log file @1 not latest in the chain but open flag still set", 1, "HY000"},
	Isc_logh_open_flag2:                   {"Log file @1 not closed properly; database recovery may be required", 1, "HY000"},
	Isc_logh_diff_dbname:                  {"Database name in the log file @1 is different", 1, "HY000"},
	Isc_logf_unexpected_eof:               {"Unexpected end of log file @1 at offset @2", 2, "HY000"},
	Isc_logr_incomplete:                   {"Incomplete log record at offset @1 in log file @2", 2, "HY000"},
	Isc_logr_header_small:                 {"Log record header too small at offset @1 in log file @2", 2, "HY000"},
	Isc_logb_small:                        {"Log block too small at offset @1 in log file @2", 2, "HY000"},
	Isc_wal_illegal_attach:                {"Illegal attempt to attach to an uninitialized WAL segment for @1", 1, "HY000"},
	Isc_wal_invalid_wpb:                   {"Invalid WAL parameter block option @1", 1, "HY000"},
	Isc_wal_err_rollover:                  {"Cannot roll over to the next log file @1", 1, "HY000"},
	Isc_no_wal:                            {"database does not use Write-ahead Log", 0, "HY000"},
	Isc_drop_wal:                          {"cannot drop log file when journaling is enabled", 0, "HY000"},
	Isc_stream_not_defined:                {"reference to invalid stream number", 0, "HY000"},
	Isc_wal_subsys_error:                  {"WAL subsystem encountered error", 0, "HY000"},
	Isc_wal_subsys_corrupt:                {"WAL subsystem corrupted", 0, "XX000"},
	Isc_no_archive:                        {"must specify archive file when enabling long term journal for databases with round-robin log files", 0, "HY000"},
	Isc_shutinprog:                        {"database @1 shutdown in progress", 1, "HY000"},
	Isc_range_in_use:                      {"refresh range number @1 already in use", 1, "42000"},
	Isc_range_not_found:                   {"refresh range number @1 not found", 1, "42000"},
	Isc_charset_not_found:                 {"CHARACTER SET @1 is not defined", 1, "2C000"},
	Isc_lock_timeout:                      {"lock time-out on wait transaction", 0, "40001"},
	Isc_prcnotdef:                         {"procedure @1 is not defined", 1, "42000"},
	Isc_prcmismat:                         {"Parameter mismatch for procedure @1", 1, "07001"},
	Isc_wal_bugcheck:                      {"Database @1: WAL subsystem bug for pid @2\n@3", 3, "XX000"},
	Isc_wal_cant_expand:                   {"Could not expand the WAL segment for database @1", 1, "HY000"},
	Isc_codnotdef:                         {"status code @1 unknown", 1, "HY000"},
	Isc_xcpnotdef:                         {"exception @1 not defined", 1, "HY000"},
	Isc_except:                            {"exception @1", 1, "HY000"},
	Isc_cache_restart:                     {"restart shared cache manager", 0, "HY013"},
	Isc_bad_lock_handle:                   {"invalid lock handle", 0, "HY000"},
	Isc_jrn_present:                       {"long-term journaling already enabled", 0, "HY000"},
	Isc_wal_err_rollover2:                 {"Unable to roll over please see Red Database log.", 0, "HY000"},
	Isc_wal_err_logwrite:                  {"WAL I/O error.  Please see Red Database log.", 0, "HY000"},
	Isc_wal_err_jrn_comm:                  {"WAL writer - Journal server communication error.  Please see Red Database log.", 0, "HY000"},
	Isc_wal_err_expansion:                 {"WAL buffers cannot be increased.  Please see Red Database log.", 0, "HY000"},
	Isc_wal_err_setup:                     {"WAL setup error.  Please see Red Database log.", 0, "HY000"},
	Isc_wal_err_ww_sync:                   {"obsolete", 0, "HY000"},
	Isc_wal_err_ww_start:                  {"Cannot start WAL writer for the database @1", 1, "HY000"},
	Isc_shutdown:                          {"database @1 shutdown", 1, "HY000"},
	Isc_existing_priv_mod:                 {"cannot modify an existing user privilege", 0, "28000"},
	Isc_primary_key_ref:                   {"Cannot delete PRIMARY KEY being used in FOREIGN KEY definition.", 0, "42000"},
	Isc_primary_key_notnull:               {"Column used in a PRIMARY constraint must be NOT NULL.", 0, "42000"},
	Isc_ref_cnstrnt_notfound:              {"Name of Referential Constraint not defined in constraints table.", 0, "42000"},
	Isc_foreign_key_notfound:              {"Non-existent PRIMARY or UNIQUE KEY specified for FOREIGN KEY.", 0, "42000"},
	Isc_ref_cnstrnt_update:                {"Cannot update constraints (RDB$REF_CONSTRAINTS).", 0, "42000"},
	Isc_check_cnstrnt_update:              {"Cannot update constraints (RDB$CHECK_CONSTRAINTS).", 0, "44000"},
	Isc_check_cnstrnt_del:                 {"Cannot delete CHECK constraint entry (RDB$CHECK_CONSTRAINTS)", 0, "44000"},
	Isc_integ_index_seg_del:               {"Cannot delete index segment used by an Integrity Constraint", 0, "42000"},
	Isc_integ_index_seg_mod:               {"Cannot update index segment used by an Integrity Constraint", 0, "42000"},
	Isc_integ_index_del:                   {"Cannot delete index used by an Integrity Constraint", 0, "42000"},
	Isc_integ_index_mod:                   {"Cannot modify index used by an Integrity Constraint", 0, "42000"},
	Isc_check_trig_del:                    {"Cannot delete trigger used by a CHECK Constraint", 0, "44000"},
	Isc_check_trig_update:                 {"Cannot update trigger used by a CHECK Constraint", 0, "44000"},
	Isc_cnstrnt_fld_del:                   {"Cannot delete column being used in an Integrity Constraint.", 0, "23000"},
	Isc_cnstrnt_fld_rename:                {"Cannot rename column being used in an Integrity Constraint.", 0, "23000"},
	Isc_rel_cnstrnt_update:                {"Cannot update constraints (RDB$RELATION_CONSTRAINTS).", 0, "42000"},
	Isc_constaint_on_view:                 {"Cannot define constraints on views", 0, "42000"},
	Isc_invld_cnstrnt_type:                {"internal Red Database consistency check (invalid RDB$CONSTRAINT_TYPE)", 0, "42000"},
	Isc_primary_key_exists:                {"Attempt to define a second PRIMARY KEY for the same table", 0, "42S11"},
	Isc_systrig_update:                    {"cannot modify or erase a system trigger", 0, "HY000"},
	Isc_not_rel_owner:                     {"only the owner of a table may reassign ownership", 0, "28000"},
	Isc_grant_obj_notfound:                {"could not find object for GRANT", 0, "01007"},
	Isc_grant_fld_notfound:                {"could not find column for GRANT", 0, "01007"},
	Isc_grant_nopriv:                      {"user does not have GRANT privileges for operation", 0, "01007"},
	Isc_nonsql_security_rel:               {"object has non-SQL security class defined", 0, "28000"},
	Isc_nonsql_security_fld:               {"column has non-SQL security class defined", 0, "28000"},
	Isc_wal_cache_err:                     {"Write-ahead Log without shared cache configuration not allowed", 0, "HY000"},
	Isc_shutfail:                          {"database shutdown unsuccessful", 0, "HY000"},
	Isc_check_constraint:                  {"Operation violates CHECK constraint @1 on view or table @2", 2, "23000"},
	Isc_bad_svc_handle:                    {"invalid service handle", 0, "HY000"},
	Isc_shutwarn:                          {"database @1 shutdown in @2 seconds", 2, "HY000"},
	Isc_wrospbver:                         {"wrong version of service parameter block", 0, "HY000"},
	Isc_bad_spb_form:                      {"unrecognized service parameter block", 0, "42000"},
	Isc_svcnotdef:                         {"service @1 is not defined", 1, "HY000"},
	Isc_no_jrn:                            {"long-term journaling not enabled", 0, "HY000"},
	Isc_transliteration_failed:            {"Cannot transliterate character between character sets", 0, "22018"},
	Isc_start_cm_for_wal:                  {"WAL defined; Cache Manager must be started first", 0, "HY000"},
	Isc_wal_ovflow_log_required:           {"Overflow log specification required for round-robin log", 0, "HY000"},
	Isc_text_subtype:                      {"Implementation of text subtype @1 not located.", 1, "42000"},
	Isc_dsql_error:                        {"Dynamic SQL Error", 0, "42000"},
	Isc_dsql_command_err:                  {"Invalid command", 0, "42000"},
	Isc_dsql_constant_err:                 {"Data type for constant unknown", 0, "HY004"},
	Isc_dsql_cursor_err:                   {"Invalid cursor reference", 0, "HY015"},
	Isc_dsql_datatype_err:                 {"Data type unknown", 0, "HY004"},
	Isc_dsql_decl_err:                     {"Invalid cursor declaration", 0, "34000"},
	Isc_dsql_cursor_update_err:            {"Cursor @1 is not updatable", 1, "24000"},
	Isc_dsql_cursor_open_err:              {"Attempt to reopen an open cursor", 0, "24000"},
	Isc_dsql_cursor_close_err:             {"Attempt to reclose a closed cursor", 0, "24000"},
	Isc_dsql_field_err:                    {"Column unknown", 0, "42S22"},
	Isc_dsql_internal_err:                 {"Internal error", 0, "XX000"},
	Isc_dsql_relation_err:                 {"Table unknown", 0, "42S02"},
	Isc_dsql_procedure_err:                {"Procedure unknown", 0, "42000"},
	Isc_dsql_request_err:                  {"Request unknown", 0, "42000"},
	Isc_dsql_sqlda_err:                    {"SQLDA error", 0, "07002"},
	Isc_dsql_var_count_err:                {"Count of read-write columns does not equal count of values", 0, "21S01"},
	Isc_dsql_stmt_handle:                  {"Invalid statement handle", 0, "HY000"},
	Isc_dsql_function_err:                 {"Function unknown", 0, "39000"},
	Isc_dsql_blob_err:                     {"Column is not a BLOB", 0, "42000"},
	Isc_collation_not_found:               {"COLLATION @1 for CHARACTER SET @2 is not defined", 2, "22021"},
	Isc_collation_not_for_charset:         {"COLLATION @1 is not valid for specified CHARACTER SET", 1, "22021"},
	Isc_dsql_dup_option:                   {"Option specified more than once", 0, "42000"},
	Isc_dsql_tran_err:                     {"Unknown transaction option", 0, "25000"},
	Isc_dsql_invalid_array:                {"Invalid array reference", 0, "HY000"},
	Isc_dsql_max_arr_dim_exceeded:         {"Array declared with too many dimensions", 0, "54000"},
	Isc_dsql_arr_range_error:              {"Illegal array dimension range", 0, "42000"},
	Isc_dsql_trigger_err:                  {"Trigger unknown", 0, "HY000"},
	Isc_dsql_subselect_err:                {"Subselect illegal in this context", 0, "42000"},
	Isc_dsql_crdb_prepare_err:             {"Cannot prepare a CREATE DATABASE statement", 0, "42000"},
	Isc_specify_field_err:                 {"must specify column name for view select expression", 0, "42000"},
	Isc_num_field_err:                     {"number of columns does not match select list", 0, "07002"},
	Isc_col_name_err:                      {"Only simple column names permitted for VIEW WITH CHECK OPTION", 0, "42000"},
	Isc_where_err:                         {"No WHERE clause for VIEW WITH CHECK OPTION", 0, "42000"},
	Isc_table_view_err:                    {"Only one table allowed for VIEW WITH CHECK OPTION", 0, "42000"},
	Isc_distinct_err:                      {"DISTINCT, GROUP or HAVING not permitted for VIEW WITH CHECK OPTION", 0, "42000"},
	Isc_key_field_count_err:               {"FOREIGN KEY column count does not match PRIMARY KEY", 0, "42000"},
	Isc_subquery_err:                      {"No subqueries permitted for VIEW WITH CHECK OPTION", 0, "42000"},
	Isc_expression_eval_err:               {"expression evaluation not supported", 0, "42000"},
	Isc_node_err:                          {"gen.c: node not supported", 0, "HY000"},
	Isc_command_end_err:                   {"Unexpected end of command", 0, "42000"},
	Isc_index_name:                        {"INDEX @1", 1, "42000"},
	Isc_exception_name:                    {"EXCEPTION @1", 1, "22000"},
	Isc_field_name:                        {"COLUMN @1", 1, "42000"},
	Isc_token_err:                         {"Token unknown", 0, "42000"},
	Isc_union_err:                         {"union not supported", 0, "42000"},
	Isc_dsql_construct_err:                {"Unsupported DSQL construct", 0, "42000"},
	Isc_field_aggregate_err:               {"column used with aggregate", 0, "42000"},
	Isc_field_ref_err:                     {"invalid column reference", 0, "42S22"},
	Isc_order_by_err:                      {"invalid ORDER BY clause", 0, "42000"},
	Isc_return_mode_err:                   {"Return mode by value not allowed for this data type", 0, "HY000"},
	Isc_extern_func_err:                   {"External functions cannot have more than 10 parameters", 0, "38000"},
	Isc_alias_conflict_err:                {"alias @1 conflicts with an alias in the same statement", 1, "42000"},
	Isc_procedure_conflict_error:          {"alias @1 conflicts with a procedure in the same statement", 1, "42000"},
	Isc_relation_conflict_err:             {"alias @1 conflicts with a table in the same statement", 1, "42000"},
	Isc_dsql_domain_err:                   {"Illegal use of keyword VALUE", 0, "42000"},
	Isc_idx_seg_err:                       {"segment count of 0 defined for index @1", 1, "42000"},
	Isc_node_name_err:                     {"A node name is not permitted in a secondary, shadow, cache or log file name", 0, "HY000"},
	Isc_table_name:                        {"TABLE @1", 1, "00000"},
	Isc_proc_name:                         {"PROCEDURE @1", 1, "42000"},
	Isc_idx_create_err:                    {"cannot create index @1", 1, "42000"},
	Isc_wal_shadow_err:                    {"Write-ahead Log with shadowing configuration not allowed", 0, "HY000"},
	Isc_dependency:                        {"there are @1 dependencies", 1, "42000"},
	Isc_idx_key_err:                       {"too many keys defined for index @1", 1, "54011"},
	Isc_dsql_file_length_err:              {"Preceding file did not specify length, so @1 must include starting page number", 1, "42000"},
	Isc_dsql_shadow_number_err:            {"Shadow number must be a positive integer", 0, "HY000"},
	Isc_dsql_token_unk_err:                {"Token unknown - line @1, column @2", 2, "42000"},
	Isc_dsql_no_relation_alias:            {"there is no alias or table named @1 at this scope level", 1, "42S02"},
	Isc_indexname:                         {"there is no index @1 for table @2", 2, "42000"},
	Isc_no_stream_plan:                    {"table or procedure @1 is not referenced in plan", 1, "HY000"},
	Isc_stream_twice:                      {"table or procedure @1 is referenced more than once in plan; use aliases to distinguish", 1, "HY000"},
	Isc_stream_not_found:                  {"table or procedure @1 is referenced in the plan but not the from list", 1, "HY000"},
	Isc_collation_requires_text:           {"Invalid use of CHARACTER SET or COLLATE", 0, "22021"},
	Isc_dsql_domain_not_found:             {"Specified domain or source column @1 does not exist", 1, "42000"},
	Isc_index_unused:                      {"index @1 cannot be used in the specified plan", 1, "42000"},
	Isc_dsql_self_join:                    {"the table @1 is referenced twice; use aliases to differentiate", 1, "42000"},
	Isc_stream_bof:                        {"attempt to fetch before the first record in a record stream", 0, "HY109"},
	Isc_stream_crack:                      {"the current position is on a crack", 0, "HY000"},
	Isc_db_or_file_exists:                 {"database or file exists", 0, "HY000"},
	Isc_invalid_operator:                  {"invalid comparison operator for find operation", 0, "42000"},
	Isc_conn_lost:                         {"Connection lost to pipe server", 0, "HY000"},
	Isc_bad_checksum:                      {"bad checksum", 0, "42000"},
	Isc_page_type_err:                     {"wrong page type", 0, "HY000"},
	Isc_ext_readonly_err:                  {"Cannot insert because the file is readonly or is on a read only medium.", 0, "42000"},
	Isc_sing_select_err:                   {"multiple rows in singleton select", 0, "21000"},
	Isc_psw_attach:                        {"cannot attach to password database", 0, "28000"},
	Isc_psw_start_trans:                   {"cannot start transaction for password database", 0, "28000"},
	Isc_invalid_direction:                 {"invalid direction for find operation", 0, "42000"},
	Isc_dsql_var_conflict:                 {"variable @1 conflicts with parameter in same procedure", 1, "HY000"},
	Isc_dsql_no_blob_array:                {"Array/BLOB/DATE data types not allowed in arithmetic", 0, "HY000"},
	Isc_dsql_base_table:                   {"@1 is not a valid base table of the specified view", 1, "42000"},
	Isc_duplicate_base_table:              {"table or procedure @1 is referenced twice in view; use an alias to distinguish", 1, "42000"},
	Isc_view_alias:                        {"view @1 has more than one base table; use aliases to distinguish", 1, "42000"},
	Isc_index_root_page_full:              {"cannot add index, index root page is full.", 0, "54000"},
	Isc_dsql_blob_type_unknown:            {"BLOB SUB_TYPE @1 is not defined", 1, "42000"},
	Isc_req_max_clones_exceeded:           {"Too many concurrent executions of the same request", 0, "54001"},
	Isc_dsql_duplicate_spec:               {"duplicate specification of @1 - not supported", 1, "42000"},
	Isc_unique_key_violation:              {"violation of PRIMARY or UNIQUE KEY constraint @1 on table @2", 2, "23000"},
	Isc_srvr_version_too_old:              {"server version too old to support all CREATE DATABASE options", 0, "HY000"},
	Isc_drdb_completed_with_errs:          {"drop database completed with errors", 0, "HY000"},
	Isc_dsql_procedure_use_err:            {"procedure @1 does not return any values", 1, "42000"},
	Isc_dsql_count_mismatch:               {"count of column list and variable list do not match", 0, "07002"},
	Isc_blob_idx_err:                      {"attempt to index BLOB column in index @1", 1, "42000"},
	Isc_array_idx_err:                     {"attempt to index array column in index @1", 1, "42000"},
	Isc_key_field_err:                     {"too few key columns found for index @1 (incorrect column name?)", 1, "42000"},
	Isc_no_delete:                         {"cannot delete", 0, "HY000"},
	Isc_del_last_field:                    {"last column in a table cannot be deleted", 0, "42000"},
	Isc_sort_err:                          {"sort error", 0, "HY000"},
	Isc_sort_mem_err:                      {"sort error: not enough memory", 0, "HY001"},
	Isc_version_err:                       {"too many versions", 0, "54000"},
	Isc_inval_key_posn:                    {"invalid key position", 0, "HY000"},
	Isc_no_segments_err:                   {"segments not allowed in expression index @1", 1, "HY000"},
	Isc_crrp_data_err:                     {"sort error: corruption in data structure", 0, "XX000"},
	Isc_rec_size_err:                      {"new record size of @1 bytes is too big", 1, "54000"},
	Isc_dsql_field_ref:                    {"Inappropriate self-reference of column", 0, "42000"},
	Isc_req_depth_exceeded:                {"request depth exceeded. (Recursive definition?)", 0, "54001"},
	Isc_no_field_access:                   {"cannot access column @1 in view @2", 2, "42S22"},
	Isc_no_dbkey:                          {"dbkey not available for multi-table views", 0, "HY000"},
	Isc_jrn_format_err:                    {"journal file wrong format", 0, "HY000"},
	Isc_jrn_file_full:                     {"intermediate journal file full", 0, "HY000"},
	Isc_dsql_open_cursor_request:          {"The prepare statement identifies a prepare statement with an open cursor", 0, "24000"},
	Isc_ib_error:                          {"Red Database error", 0, "HY000"},
	Isc_cache_redef:                       {"Cache redefined", 0, "HY013"},
	Isc_cache_too_small:                   {"Insufficient memory to allocate page buffer cache", 0, "HY013"},
	Isc_log_redef:                         {"Log redefined", 0, "HY000"},
	Isc_log_too_small:                     {"Log size too small", 0, "HY000"},
	Isc_partition_too_small:               {"Log partition size too small", 0, "HY000"},
	Isc_partition_not_supp:                {"Partitions not supported in series of log file specification", 0, "HY000"},
	Isc_log_length_spec:                   {"Total length of a partitioned log must be specified", 0, "HY000"},
	Isc_precision_err:                     {"Precision must be from 1 to 18", 0, "HY104"},
	Isc_scale_nogt:                        {"Scale must be between zero and precision", 0, "HY104"},
	Isc_expec_short:                       {"Short integer expected", 0, "22003"},
	Isc_expec_long:                        {"Long integer expected", 0, "22003"},
	Isc_expec_ushort:                      {"Unsigned short integer expected", 0, "22003"},
	Isc_escape_invalid:                    {"Invalid ESCAPE sequence", 0, "22025"},
	Isc_svcnoexe:                          {"service @1 does not have an associated executable", 1, "HY000"},
	Isc_net_lookup_err:                    {"Failed to locate host machine.", 0, "08006"},
	Isc_service_unknown:                   {"Undefined service @1/@2.", 2, "HY000"},
	Isc_host_unknown:                      {"The specified name was not found in the hosts file or Domain Name Services.", 0, "08001"},
	Isc_grant_nopriv_on_base:              {"user does not have GRANT privileges on base table/view for operation", 0, "01007"},
	Isc_dyn_fld_ambiguous:                 {"Ambiguous column reference.", 0, "42702"},
	Isc_dsql_agg_ref_err:                  {"Invalid aggregate reference", 0, "42000"},
	Isc_complex_view:                      {"navigational stream @1 references a view with more than one base table", 1, "HY000"},
	Isc_unprepared_stmt:                   {"Attempt to execute an unprepared dynamic SQL statement.", 0, "HY007"},
	Isc_expec_positive:                    {"Positive value expected", 0, "42000"},
	Isc_dsql_sqlda_value_err:              {"Incorrect values within SQLDA structure", 0, "HY000"},
	Isc_invalid_array_id:                  {"invalid blob id", 0, "HY000"},
	Isc_extfile_uns_op:                    {"Operation not supported for EXTERNAL FILE table @1", 1, "42000"},
	Isc_svc_in_use:                        {"Service is currently busy: @1", 1, "HY000"},
	Isc_err_stack_limit:                   {"stack size insufficient to execute current request", 0, "HY013"},
	Isc_invalid_key:                       {"Invalid key for find operation", 0, "42000"},
	Isc_net_init_error:                    {"Error initializing the network software.", 0, "08006"},
	Isc_loadlib_failure:                   {"Unable to load required library @1.", 1, "HY000"},
	Isc_network_error:                     {"Unable to complete network request to host \"@1\".", 1, "08006"},
	Isc_net_connect_err:                   {"Failed to establish a connection.", 0, "08006"},
	Isc_net_connect_listen_err:            {"Error while listening for an incoming connection.", 0, "08006"},
	Isc_net_event_connect_err:             {"Failed to establish a secondary connection for event processing.", 0, "08006"},
	Isc_net_event_listen_err:              {"Error while listening for an incoming event connection request.", 0, "08006"},
	Isc_net_read_err:                      {"Error reading data from the connection.", 0, "08006"},
	Isc_net_write_err:                     {"Error writing data to the connection.", 0, "08006"},
	Isc_integ_index_deactivate:            {"Cannot deactivate index used by an integrity constraint", 0, "42000"},
	Isc_integ_deactivate_primary:          {"Cannot deactivate index used by a PRIMARY/UNIQUE constraint", 0, "42000"},
	Isc_cse_not_supported:                 {"Client/Server Express not supported in this release", 0, "HY000"},
	Isc_tra_must_sweep:                    {"", 0, "HY000"},
	Isc_unsupported_network_drive:         {"Access to databases on file servers is not supported.", 0, "HY000"},
	Isc_io_create_err:                     {"Error while trying to create file", 0, "08001"},
	Isc_io_open_err:                       {"Error while trying to open file", 0, "08001"},
	Isc_io_close_err:                      {"Error while trying to close file", 0, "HY000"},
	Isc_io_read_err:                       {"Error while trying to read from file", 0, "HY000"},
	Isc_io_write_err:                      {"Error while trying to write to file", 0, "HY000"},
	Isc_io_delete_err:                     {"Error while trying to delete file", 0, "HY000"},
	Isc_io_access_err:                     {"Error while trying to access file", 0, "HY000"},
	Isc_udf_exception:                     {"A fatal exception occurred during the execution of a user defined function.", 0, "38000"},
	Isc_lost_db_connection:                {"connection lost to database", 0, "08006"},
	Isc_no_write_user_priv:                {"User cannot write to RDB$USER_PRIVILEGES", 0, "28000"},
	Isc_token_too_long:                    {"token size exceeds limit", 0, "42000"},
	Isc_max_att_exceeded:                  {"Maximum user count exceeded.  Contact your database administrator.", 0, "08001"},
	Isc_login_same_as_role_name:           {"Your login @1 is same as one of the SQL role name. Ask your database administrator to set up a valid Red Database login.", 1, "28000"},
	Isc_reftable_requires_pk:              {"\"REFERENCES table\" without \"(column)\" requires PRIMARY KEY on referenced table", 0, "42000"},
	Isc_usrname_too_long:                  {"The username entered is too long.  Maximum length is 31 bytes.", 0, "28000"},
	Isc_password_too_long:                 {"The password specified is too long.  Maximum length is 8 bytes.", 0, "28000"},
	Isc_usrname_required:                  {"A username is required for this operation.", 0, "28000"},
	Isc_password_required:                 {"A password is required for this operation", 0, "28000"},
	Isc_bad_protocol:                      {"The network protocol specified is invalid", 0, "42000"},
	Isc_dup_usrname_found:                 {"A duplicate user name was found in the security database", 0, "28000"},
	Isc_usrname_not_found:                 {"The user name specified was not found in the security database", 0, "28000"},
	Isc_error_adding_sec_record:           {"An error occurred while attempting to add the user.", 0, "28000"},
	Isc_error_modifying_sec_record:        {"An error occurred while attempting to modify the user record.", 0, "28000"},
	Isc_error_deleting_sec_record:         {"An error occurred while attempting to delete the user record.", 0, "28000"},
	Isc_error_updating_sec_db:             {"An error occurred while updating the security database.", 0, "28000"},
	Isc_sort_rec_size_err:                 {"sort record size of @1 bytes is too big", 1, "HY000"},
	Isc_bad_default_value:                 {"can not define a not null column with NULL as default value", 0, "42000"},
	Isc_invalid_clause:                    {"invalid clause --- '@1'", 1, "42000"},
	Isc_too_many_handles:                  {"too many open handles to database", 0, "HY000"},
	Isc_optimizer_blk_exc:                 {"size of optimizer block exceeded", 0, "HY000"},
	Isc_invalid_string_constant:           {"a string constant is delimited by double quotes", 0, "42000"},
	Isc_transitional_date:                 {"DATE must be changed to TIMESTAMP", 0, "HY000"},
	Isc_read_only_database:                {"attempted update on read-only database", 0, "42000"},
	Isc_must_be_dialect_2_and_up:          {"SQL dialect @1 is not supported in this database", 1, "HY000"},
	Isc_blob_filter_exception:             {"A fatal exception occurred during the execution of a blob filter.", 0, "HY000"},
	Isc_exception_access_violation:        {"Access violation.  The code attempted to access a virtual address without privilege to do so.", 0, "HY000"},
	Isc_exception_datatype_missalignment:  {"Datatype misalignment.  The attempted to read or write a value that was not stored on a memory boundary.", 0, "22000"},
	Isc_exception_array_bounds_exceeded:   {"Array bounds exceeded.  The code attempted to access an array element that is out of bounds.", 0, "22000"},
	Isc_exception_float_denormal_operand:  {"Float denormal operand.  One of the floating-point operands is too small to represent a standard float value.", 0, "22000"},
	Isc_exception_float_divide_by_zero:    {"Floating-point divide by zero.  The code attempted to divide a floating-point value by zero.", 0, "22012"},
	Isc_exception_float_inexact_result:    {"Floating-point inexact result.  The result of a floating-point operation cannot be represented as a decimal fraction.", 0, "22000"},
	Isc_exception_float_invalid_operand:   {"Floating-point invalid operand.  An indeterminant error occurred during a floating-point operation.", 0, "22000"},
	Isc_exception_float_overflow:          {"Floating-point overflow.  The exponent of a floating-point operation is greater than the magnitude allowed.", 0, "22003"},
	Isc_exception_float_stack_check:       {"Floating-point stack check.  The stack overflowed or underflowed as the result of a floating-point operation.", 0, "22000"},
	Isc_exception_float_underflow:         {"Floating-point underflow.  The exponent of a floating-point operation is less than the magnitude allowed.", 0, "22003"},
	Isc_exception_integer_divide_by_zero:  {"Integer divide by zero.  The code attempted to divide an integer value by an integer divisor of zero.", 0, "22012"},
	Isc_exception_integer_overflow:        {"Integer overflow.  The result of an integer operation caused the most significant bit of the result to carry.", 0, "22003"},
	Isc_exception_unknown:                 {"An exception occurred that does not have a description.  Exception number @1.", 1, "HY000"},
	Isc_exception_stack_overflow:          {"Stack overflow.  The resource requirements of the runtime stack have exceeded the memory available to it.", 0, "HY001"},
	Isc_exception_sigsegv:                 {"Segmentation Fault. The code attempted to access memory without privileges.", 0, "HY000"},
	Isc_exception_sigill:                  {"Illegal Instruction. The Code attempted to perform an illegal operation.", 0, "HY000"},
	Isc_exception_sigbus:                  {"Bus Error. The Code caused a system bus error.", 0, "HY000"},
	Isc_exception_sigfpe:                  {"Floating Point Error. The Code caused an Arithmetic Exception or a floating point exception.", 0, "HY000"},
	Isc_ext_file_delete:                   {"Cannot delete rows from external files.", 0, "42000"},
	Isc_ext_file_modify:                   {"Cannot update rows in external files.", 0, "42000"},
	Isc_adm_task_denied:                   {"Unable to perform operation", 0, "28000"},
	Isc_extract_input_mismatch:            {"Specified EXTRACT part does not exist in input datatype", 0, "42000"},
	Isc_insufficient_svc_privileges:       {"Service @1 requires SYSDBA permissions.  Reattach to the Service Manager using the SYSDBA account.", 1, "28000"},
	Isc_file_in_use:                       {"The file @1 is currently in use by another process.  Try again later.", 1, "HY000"},
	Isc_service_att_err:                   {"Cannot attach to services manager", 0, "HY000"},
	Isc_ddl_not_allowed_by_db_sql_dial:    {"Metadata update statement is not allowed by the current database SQL dialect @1", 1, "42000"},
	Isc_cancelled:                         {"operation was cancelled", 0, "HY008"},
	Isc_unexp_spb_form:                    {"unexpected item in service parameter block, expected @1", 1, "HY000"},
	Isc_sql_dialect_datatype_unsupport:    {"Client SQL dialect @1 does not support reference to @2 datatype", 2, "HY000"},
	Isc_svcnouser:                         {"user name and password are required while attaching to the services manager", 0, "HY000"},
	Isc_depend_on_uncommitted_rel:         {"You created an indirect dependency on uncommitted metadata. You must roll back the current transaction.", 0, "42000"},
	Isc_svc_name_missing:                  {"The service name was not specified.", 0, "HY000"},
	Isc_too_many_contexts:                 {"Too many Contexts of Relation/Procedure/Views. Maximum allowed is 256", 0, "54001"},
	Isc_datype_notsup:                     {"data type not supported for arithmetic", 0, "22008"},
	Isc_dialect_reset_warning:             {"Database dialect being changed from 3 to 1", 0, "01000"},
	Isc_dialect_not_changed:               {"Database dialect not changed.", 0, "HY000"},
	Isc_database_create_failed:            {"Unable to create database @1", 1, "HY000"},
	Isc_inv_dialect_specified:             {"Database dialect @1 is not a valid dialect.", 1, "42000"},
	Isc_valid_db_dialects:                 {"Valid database dialects are @1.", 1, "00000"},
	Isc_sqlwarn:                           {"SQL warning code = @1", 1, "00000"},
	Isc_dtype_renamed:                     {"DATE data type is now called TIMESTAMP", 0, "HY000"},
	Isc_extern_func_dir_error:             {"Function @1 is in @2, which is not in a permitted directory for external functions.", 2, "38000"},
	Isc_date_range_exceeded:               {"value exceeds the range for valid dates", 0, "22008"},
	Isc_inv_client_dialect_specified:      {"passed client dialect @1 is not a valid dialect.", 1, "42000"},
	Isc_valid_client_dialects:             {"Valid client dialects are @1.", 1, "00000"},
	Isc_optimizer_between_err:             {"Unsupported field type specified in BETWEEN predicate.", 0, "HY004"},
	Isc_service_not_supported:             {"Services functionality will be supported in a later version  of the product", 0, "HY000"},
	Isc_generator_name:                    {"GENERATOR @1", 1, "00000"},
	Isc_udf_name:                          {"Function @1", 1, "38000"},
	Isc_bad_limit_param:                   {"Invalid parameter to FETCH or FIRST. Only integers >= 0 are allowed.", 0, "HY000"},
	Isc_bad_skip_param:                    {"Invalid parameter to OFFSET or SKIP. Only integers >= 0 are allowed.", 0, "42000"},
	Isc_io_32bit_exceeded_err:             {"File exceeded maximum size of 2GB.  Add another database file or use a 64 bit I/O version of Red Database.", 0, "54000"},
	Isc_invalid_savepoint:                 {"Unable to find savepoint with name @1 in transaction context", 1, "3B000"},
	Isc_dsql_column_pos_err:               {"Invalid column position used in the @1 clause", 1, "42000"},
	Isc_dsql_agg_where_err:                {"Cannot use an aggregate or window function in a WHERE clause, use HAVING (for aggregate only) instead", 0, "42000"},
	Isc_dsql_agg_group_err:                {"Cannot use an aggregate or window function in a GROUP BY clause", 0, "42000"},
	Isc_dsql_agg_column_err:               {"Invalid expression in the @1 (not contained in either an aggregate function or the GROUP BY clause)", 1, "42000"},
	Isc_dsql_agg_having_err:               {"Invalid expression in the @1 (neither an aggregate function nor a part of the GROUP BY clause)", 1, "42000"},
	Isc_dsql_agg_nested_err:               {"Nested aggregate and window functions are not allowed", 0, "42000"},
	Isc_exec_sql_invalid_arg:              {"Invalid argument in EXECUTE STATEMENT - cannot convert to string", 0, "42000"},
	Isc_exec_sql_invalid_req:              {"Wrong request type in EXECUTE STATEMENT '@1'", 1, "42000"},
	Isc_exec_sql_invalid_var:              {"Variable type (position @1) in EXECUTE STATEMENT '@2' INTO does not match returned column type", 2, "42000"},
	Isc_exec_sql_max_call_exceeded:        {"Too many recursion levels of EXECUTE STATEMENT", 0, "42000"},
	Isc_conf_access_denied:                {"Use of @1 at location @2 is not allowed by server configuration", 2, "28000"},
	Isc_wrong_backup_state:                {"Cannot change difference file name while database is in backup mode", 0, "HY000"},
	Isc_wal_backup_err:                    {"Physical backup is not allowed while Write-Ahead Log is in use", 0, "HY000"},
	Isc_cursor_not_open:                   {"Cursor is not open", 0, "24000"},
	Isc_bad_shutdown_mode:                 {"Target shutdown mode is invalid for database \"@1\"", 1, "08007"},
	Isc_concat_overflow:                   {"Concatenation overflow. Resulting string cannot exceed 32765 bytes in length.", 0, "HY000"},
	Isc_bad_substring_offset:              {"Invalid offset parameter @1 to SUBSTRING. Only positive integers are allowed.", 1, "22011"},
	Isc_foreign_key_target_doesnt_exist:   {"Foreign key reference target does not exist", 0, "42000"},
	Isc_foreign_key_references_present:    {"Foreign key references are present for the record", 0, "42000"},
	Isc_no_update:                         {"cannot update", 0, "HY000"},
	Isc_cursor_already_open:               {"Cursor is already open", 0, "24000"},
	Isc_stack_trace:                       {"@1", 1, "00000"},
	Isc_ctx_var_not_found:                 {"Context variable '@1' is not found in namespace '@2'", 2, "HY000"},
	Isc_ctx_namespace_invalid:             {"Invalid namespace name '@1' passed to @2", 2, "HY000"},
	Isc_ctx_too_big:                       {"Too many context variables", 0, "54000"},
	Isc_ctx_bad_argument:                  {"Invalid argument passed to @1", 1, "HY000"},
	Isc_identifier_too_long:               {"BLR syntax error. Identifier @1... is too long", 1, "42000"},
	Isc_except2:                           {"exception @1", 1, "HY000"},
	Isc_malformed_string:                  {"Malformed string", 0, "22000"},
	Isc_prc_out_param_mismatch:            {"Output parameter mismatch for procedure @1", 1, "07002"},
	Isc_command_end_err2:                  {"Unexpected end of command - line @1, column @2", 2, "42000"},
	Isc_partner_idx_incompat_type:         {"partner index segment no @1 has incompatible data type", 1, "42000"},
	Isc_bad_substring_length:              {"Invalid length parameter @1 to SUBSTRING. Negative integers are not allowed.", 1, "22011"},
	Isc_charset_not_installed:             {"CHARACTER SET @1 is not installed", 1, "2C000"},
	Isc_collation_not_installed:           {"COLLATION @1 for CHARACTER SET @2 is not installed", 2, "22021"},
	Isc_att_shutdown:                      {"connection shutdown", 0, "08003"},
	Isc_blobtoobig:                        {"Maximum BLOB size exceeded", 0, "54000"},
	Isc_must_have_phys_field:              {"Can't have relation with only computed fields or constraints", 0, "42000"},
	Isc_invalid_time_precision:            {"Time precision exceeds allowed range (0-@1)", 1, "22007"},
	Isc_blob_convert_error:                {"Unsupported conversion to target type BLOB (subtype @1)", 1, "HY000"},
	Isc_array_convert_error:               {"Unsupported conversion to target type ARRAY", 0, "22000"},
	Isc_record_lock_not_supp:              {"Stream does not support record locking", 0, "40001"},
	Isc_partner_idx_not_found:             {"Cannot create foreign key constraint @1. Partner index does not exist or is inactive.", 1, "42000"},
	Isc_tra_num_exc:                       {"Transactions count exceeded. Perform backup and restore to make database operable again", 0, "54000"},
	Isc_field_disappeared:                 {"Column has been unexpectedly deleted", 0, "42000"},
	Isc_met_wrong_gtt_scope:               {"@1 cannot depend on @2", 2, "HY000"},
	Isc_subtype_for_internal_use:          {"Blob sub_types bigger than 1 (text) are for internal use only", 0, "HY000"},
	Isc_illegal_prc_type:                  {"Procedure @1 is not selectable (it does not contain a SUSPEND statement)", 1, "42000"},
	Isc_invalid_sort_datatype:             {"Datatype @1 is not supported for sorting operation", 1, "42000"},
	Isc_collation_name:                    {"COLLATION @1", 1, "00000"},
	Isc_domain_name:                       {"DOMAIN @1", 1, "00000"},
	Isc_domnotdef:                         {"domain @1 is not defined", 1, "42000"},
	Isc_array_max_dimensions:              {"Array data type can use up to @1 dimensions", 1, "54000"},
	Isc_max_db_per_trans_allowed:          {"A multi database transaction cannot span more than @1 databases", 1, "54000"},
	Isc_bad_debug_format:                  {"Bad debug info format", 0, "00000"},
	Isc_bad_proc_BLR:                      {"Error while parsing procedure @1's BLR", 1, "2F000"},
	Isc_key_too_big:                       {"index key too big", 0, "54000"},
	Isc_concurrent_transaction:            {"concurrent transaction number is @1", 1, "HY000"},
	Isc_not_valid_for_var:                 {"validation error for variable @1, value \"@2\"", 2, "42000"},
	Isc_not_valid_for:                     {"validation error for @1, value \"@2\"", 2, "42000"},
	Isc_need_difference:                   {"Difference file name should be set explicitly for database on raw device", 0, "HY000"},
	Isc_long_login:                        {"Login name too long (@1 characters, maximum allowed @2)", 2, "HY000"},
	Isc_fldnotdef2:                        {"column @1 is not defined in procedure @2", 2, "42S22"},
	Isc_invalid_similar_pattern:           {"Invalid SIMILAR TO pattern", 0, "42000"},
	Isc_bad_teb_form:                      {"Invalid TEB format", 0, "HY000"},
	Isc_tpb_multiple_txn_isolation:        {"Found more than one transaction isolation in TPB", 0, "HY000"},
	Isc_tpb_reserv_before_table:           {"Table reservation lock type @1 requires table name before in TPB", 1, "HY000"},
	Isc_tpb_multiple_spec:                 {"Found more than one @1 specification in TPB", 1, "HY000"},
	Isc_tpb_option_without_rc:             {"Option @1 requires READ COMMITTED isolation in TPB", 1, "HY000"},
	Isc_tpb_conflicting_options:           {"Option @1 is not valid if @2 was used previously in TPB", 2, "HY000"},
	Isc_tpb_reserv_missing_tlen:           {"Table name length missing after table reservation @1 in TPB", 1, "HY000"},
	Isc_tpb_reserv_long_tlen:              {"Table name length @1 is too long after table reservation @2 in TPB", 2, "HY000"},
	Isc_tpb_reserv_missing_tname:          {"Table name length @1 without table name after table reservation @2 in TPB", 2, "HY000"},
	Isc_tpb_reserv_corrup_tlen:            {"Table name length @1 goes beyond the remaining TPB size after table reservation @2", 2, "HY000"},
	Isc_tpb_reserv_null_tlen:              {"Table name length is zero after table reservation @1 in TPB", 1, "HY000"},
	Isc_tpb_reserv_relnotfound:            {"Table or view @1 not defined in system tables after table reservation @2 in TPB", 2, "HY000"},
	Isc_tpb_reserv_baserelnotfound:        {"Base table or view @1 for view @2 not defined in system tables after table reservation @3 in TPB", 3, "HY000"},
	Isc_tpb_missing_len:                   {"Option length missing after option @1 in TPB", 1, "HY000"},
	Isc_tpb_missing_value:                 {"Option length @1 without value after option @2 in TPB", 2, "HY000"},
	Isc_tpb_corrupt_len:                   {"Option length @1 goes beyond the remaining TPB size after option @2", 2, "HY000"},
	Isc_tpb_null_len:                      {"Option length is zero after table reservation @1 in TPB", 1, "HY000"},
	Isc_tpb_overflow_len:                  {"Option length @1 exceeds the range for option @2 in TPB", 2, "HY000"},
	Isc_tpb_invalid_value:                 {"Option value @1 is invalid for the option @2 in TPB", 2, "HY000"},
	Isc_tpb_reserv_stronger_wng:           {"Preserving previous table reservation @1 for table @2, stronger than new @3 in TPB", 3, "HY000"},
	Isc_tpb_reserv_stronger:               {"Table reservation @1 for table @2 already specified and is stronger than new @3 in TPB", 3, "HY000"},
	Isc_tpb_reserv_max_recursion:          {"Table reservation reached maximum recursion of @1 when expanding views in TPB", 1, "HY000"},
	Isc_tpb_reserv_virtualtbl:             {"Table reservation in TPB cannot be applied to @1 because it's a virtual table", 1, "HY000"},
	Isc_tpb_reserv_systbl:                 {"Table reservation in TPB cannot be applied to @1 because it's a system table", 1, "HY000"},
	Isc_tpb_reserv_temptbl:                {"Table reservation @1 or @2 in TPB cannot be applied to @3 because it's a temporary table", 3, "HY000"},
	Isc_tpb_readtxn_after_writelock:       {"Cannot set the transaction in read only mode after a table reservation isc_tpb_lock_write in TPB", 0, "HY000"},
	Isc_tpb_writelock_after_readtxn:       {"Cannot take a table reservation isc_tpb_lock_write in TPB because the transaction is in read only mode", 0, "HY000"},
	Isc_time_range_exceeded:               {"value exceeds the range for a valid time", 0, "22008"},
	Isc_datetime_range_exceeded:           {"value exceeds the range for valid timestamps", 0, "22008"},
	Isc_string_truncation:                 {"string right truncation", 0, "22001"},
	Isc_blob_truncation:                   {"blob truncation when converting to a string: length limit exceeded", 0, "22001"},
	Isc_numeric_out_of_range:              {"numeric value is out of range", 0, "22003"},
	Isc_shutdown_timeout:                  {"Red Database shutdown is still in progress after the specified timeout", 0, "HYT00"},
	Isc_att_handle_busy:                   {"Attachment handle is busy", 0, "HY010"},
	Isc_bad_udf_freeit:                    {"Bad written UDF detected: pointer returned in FREE_IT function was not allocated by ib_util_malloc", 0, "39000"},
	Isc_eds_provider_not_found:            {"External Data Source provider '@1' not found", 1, "42000"},
	Isc_eds_connection:                    {"Execute statement error at @1 :\n@2Data source : @3", 3, "42000"},
	Isc_eds_preprocess:                    {"Execute statement preprocess SQL error", 0, "42000"},
	Isc_eds_stmt_expected:                 {"Statement expected", 0, "42000"},
	Isc_eds_prm_name_expected:             {"Parameter name expected", 0, "42000"},
	Isc_eds_unclosed_comment:              {"Unclosed comment found near '@1'", 1, "42000"},
	Isc_eds_statement:                     {"Execute statement error at @1 :\n@2Statement : @3\nData source : @4", 4, "42000"},
	Isc_eds_input_prm_mismatch:            {"Input parameters mismatch", 0, "42000"},
	Isc_eds_output_prm_mismatch:           {"Output parameters mismatch", 0, "42000"},
	Isc_eds_input_prm_not_set:             {"Input parameter '@1' have no value set", 1, "42000"},
	Isc_too_big_blr:                       {"BLR stream length @1 exceeds implementation limit @2", 2, "HY000"},
	Isc_montabexh:                         {"Monitoring table space exhausted", 0, "HY001"},
	Isc_modnotfound:                       {"module name or entrypoint could not be found", 0, "39000"},
	Isc_nothing_to_cancel:                 {"nothing to cancel", 0, "HY018"},
	Isc_ibutil_not_loaded:                 {"ib_util library has not been loaded to deallocate memory returned by FREE_IT function", 0, "39000"},
	Isc_circular_computed:                 {"Cannot have circular dependencies with computed fields", 0, "42000"},
	Isc_psw_db_error:                      {"Security database error", 0, "28000"},
	Isc_invalid_type_datetime_op:          {"Invalid data type in DATE/TIME/TIMESTAMP addition or subtraction in add_datettime()", 0, "42000"},
	Isc_onlycan_add_timetodate:            {"Only a TIME value can be added to a DATE value", 0, "42000"},
	Isc_onlycan_add_datetotime:            {"Only a DATE value can be added to a TIME value", 0, "42000"},
	Isc_onlycansub_tstampfromtstamp:       {"TIMESTAMP values can be subtracted only from another TIMESTAMP value", 0, "42000"},
	Isc_onlyoneop_mustbe_tstamp:           {"Only one operand can be of type TIMESTAMP", 0, "42000"},
	Isc_invalid_extractpart_time:          {"Only HOUR, MINUTE, SECOND and MILLISECOND can be extracted from TIME values", 0, "42000"},
	Isc_invalid_extractpart_date:          {"HOUR, MINUTE, SECOND and MILLISECOND cannot be extracted from DATE values", 0, "42000"},
	Isc_invalidarg_extract:                {"Invalid argument for EXTRACT() not being of DATE/TIME/TIMESTAMP type", 0, "42000"},
	Isc_sysf_argmustbe_exact:              {"Arguments for @1 must be integral types or NUMERIC/DECIMAL without scale", 1, "42000"},
	Isc_sysf_argmustbe_exact_or_fp:        {"First argument for @1 must be integral type or floating point type", 1, "42000"},
	Isc_sysf_argviolates_uuidtype:         {"Human readable UUID argument for @1 must be of string type", 1, "42000"},
	Isc_sysf_argviolates_uuidlen:          {"Human readable UUID argument for @2 must be of exact length @1", 2, "42000"},
	Isc_sysf_argviolates_uuidfmt:          {"Human readable UUID argument for @3 must have \"-\" at position @2 instead of \"@1\"", 3, "42000"},
	Isc_sysf_argviolates_guidigits:        {"Human readable UUID argument for @3 must have hex digit at position @2 instead of \"@1\"", 3, "42000"},
	Isc_sysf_invalid_addpart_time:         {"Only HOUR, MINUTE, SECOND and MILLISECOND can be added to TIME values in @1", 1, "42000"},
	Isc_sysf_invalid_add_datetime:         {"Invalid data type in addition of part to DATE/TIME/TIMESTAMP in @1", 1, "42000"},
	Isc_sysf_invalid_addpart_dtime:        {"Invalid part @1 to be added to a DATE/TIME/TIMESTAMP value in @2", 2, "42000"},
	Isc_sysf_invalid_add_dtime_rc:         {"Expected DATE/TIME/TIMESTAMP type in evlDateAdd() result", 0, "42000"},
	Isc_sysf_invalid_diff_dtime:           {"Expected DATE/TIME/TIMESTAMP type as first and second argument to @1", 1, "42000"},
	Isc_sysf_invalid_timediff:             {"The result of TIME-<value> in @1 cannot be expressed in YEAR, MONTH, DAY or WEEK", 1, "42000"},
	Isc_sysf_invalid_tstamptimediff:       {"The result of TIME-TIMESTAMP or TIMESTAMP-TIME in @1 cannot be expressed in HOUR, MINUTE, SECOND or MILLISECOND", 1, "42000"},
	Isc_sysf_invalid_datetimediff:         {"The result of DATE-TIME or TIME-DATE in @1 cannot be expressed in HOUR, MINUTE, SECOND and MILLISECOND", 1, "42000"},
	Isc_sysf_invalid_diffpart:             {"Invalid part @1 to express the difference between two DATE/TIME/TIMESTAMP values in @2", 2, "42000"},
	Isc_sysf_argmustbe_positive:           {"Argument for @1 must be positive", 1, "42000"},
	Isc_sysf_basemustbe_positive:          {"Base for @1 must be positive", 1, "42000"},
	Isc_sysf_argnmustbe_nonneg:            {"Argument #@1 for @2 must be zero or positive", 2, "42000"},
	Isc_sysf_argnmustbe_positive:          {"Argument #@1 for @2 must be positive", 2, "42000"},
	Isc_sysf_invalid_zeropowneg:           {"Base for @1 cannot be zero if exponent is negative", 1, "42000"},
	Isc_sysf_invalid_negpowfp:             {"Base for @1 cannot be negative if exponent is not an integral value", 1, "42000"},
	Isc_sysf_invalid_scale:                {"The numeric scale must be between -128 and 127 in @1", 1, "42000"},
	Isc_sysf_argmustbe_nonneg:             {"Argument for @1 must be zero or positive", 1, "42000"},
	Isc_sysf_binuuid_mustbe_str:           {"Binary UUID argument for @1 must be of string type", 1, "42000"},
	Isc_sysf_binuuid_wrongsize:            {"Binary UUID argument for @2 must use @1 bytes", 2, "42000"},
	Isc_missing_required_spb:              {"Missing required item @1 in service parameter block", 1, "HY000"},
	Isc_net_server_shutdown:               {"@1 server is shutdown", 1, "08006"},
	Isc_bad_conn_str:                      {"Invalid connection string", 0, "08001"},
	Isc_bad_epb_form:                      {"Unrecognized events block", 0, "HY000"},
	Isc_no_threads:                        {"Could not start first worker thread - shutdown server", 0, "08004"},
	Isc_net_event_connect_timeout:         {"Timeout occurred while waiting for a secondary connection for event processing", 0, "08006"},
	Isc_sysf_argmustbe_nonzero:            {"Argument for @1 must be different than zero", 1, "42000"},
	Isc_sysf_argmustbe_range_inc1_1:       {"Argument for @1 must be in the range [-1, 1]", 1, "42000"},
	Isc_sysf_argmustbe_gteq_one:           {"Argument for @1 must be greater or equal than one", 1, "42000"},
	Isc_sysf_argmustbe_range_exc1_1:       {"Argument for @1 must be in the range ]-1, 1[", 1, "42000"},
	Isc_internal_rejected_params:          {"Incorrect parameters provided to internal function @1", 1, "HY000"},
	Isc_sysf_fp_overflow:                  {"Floating point overflow in built-in function @1", 1, "42000"},
	Isc_udf_fp_overflow:                   {"Floating point overflow in result from UDF @1", 1, "39000"},
	Isc_udf_fp_nan:                        {"Invalid floating point value returned by UDF @1", 1, "39000"},
	Isc_instance_conflict:                 {"Shared memory area is probably already created by another engine instance in another Windows session", 0, "42000"},
	Isc_out_of_temp_space:                 {"No free space found in temporary directories", 0, "HY000"},
	Isc_eds_expl_tran_ctrl:                {"Explicit transaction control is not allowed", 0, "42000"},
	Isc_no_trusted_spb:                    {"Use of TRUSTED switches in spb_command_line is prohibited", 0, "28000"},
	Isc_package_name:                      {"PACKAGE @1", 1, "42000"},
	Isc_cannot_make_not_null:              {"Cannot make field @1 of table @2 NOT NULL because there are NULLs present", 2, "22006"},
	Isc_feature_removed:                   {"Feature @1 is not supported anymore", 1, "0A000"},
	Isc_view_name:                         {"VIEW @1", 1, "00000"},
	Isc_lock_dir_access:                   {"Can not access lock files directory @1", 1, "08006"},
	Isc_invalid_fetch_option:              {"Fetch option @1 is invalid for a non-scrollable cursor", 1, "HY106"},
	Isc_bad_fun_BLR:                       {"Error while parsing function @1's BLR", 1, "2F000"},
	Isc_func_pack_not_implemented:         {"Cannot execute function @1 of the unimplemented package @2", 2, "2F000"},
	Isc_proc_pack_not_implemented:         {"Cannot execute procedure @1 of the unimplemented package @2", 2, "2F000"},
	Isc_eem_func_not_returned:             {"External function @1 not returned by the external engine plugin @2", 2, "39000"},
	Isc_eem_proc_not_returned:             {"External procedure @1 not returned by the external engine plugin @2", 2, "39000"},
	Isc_eem_trig_not_returned:             {"External trigger @1 not returned by the external engine plugin @2", 2, "39000"},
	Isc_eem_bad_plugin_ver:                {"Incompatible plugin version @1 for external engine @2", 2, "39000"},
	Isc_eem_engine_notfound:               {"External engine @1 not found", 1, "39000"},
	Isc_attachment_in_use:                 {"Attachment is in use", 0, "08002"},
	Isc_transaction_in_use:                {"Transaction is in use", 0, "25S02"},
	Isc_pman_cannot_load_plugin:           {"Error loading plugin @1", 1, "39000"},
	Isc_pman_module_notfound:              {"Loadable module @1 not found", 1, "00000"},
	Isc_pman_entrypoint_notfound:          {"Standard plugin entrypoint does not exist in module @1", 1, "00000"},
	Isc_pman_module_bad:                   {"Module @1 exists but can not be loaded", 1, "00000"},
	Isc_pman_plugin_notfound:              {"Module @1 does not contain plugin @2 type @3", 3, "00000"},
	Isc_sysf_invalid_trig_namespace:       {"Invalid usage of context namespace DDL_TRIGGER", 0, "42000"},
	Isc_unexpected_null:                   {"Value is NULL but isNull parameter was not informed", 0, "22002"},
	Isc_type_notcompat_blob:               {"Type @1 is incompatible with BLOB", 1, "2200G"},
	Isc_invalid_date_val:                  {"Invalid date", 0, "22003"},
	Isc_invalid_time_val:                  {"Invalid time", 0, "22003"},
	Isc_invalid_timestamp_val:             {"Invalid timestamp", 0, "22003"},
	Isc_invalid_index_val:                 {"Invalid index @1 in function @2", 2, "22003"},
	Isc_formatted_exception:               {"@1", 1, "00000"},
	Isc_async_active:                      {"Asynchronous call is already running for this attachment", 0, "HY018"},
	Isc_private_function:                  {"Function @1 is private to package @2", 2, "42000"},
	Isc_private_procedure:                 {"Procedure @1 is private to package @2", 2, "42000"},
	Isc_request_outdated:                  {"Request can't access new records in relation @1 and should be recompiled", 1, "24000"},
	Isc_bad_events_handle:                 {"invalid events id (handle)", 0, "HY000"},
	Isc_cannot_copy_stmt:                  {"Cannot copy statement @1", 1, "XX000"},
	Isc_invalid_boolean_usage:             {"Invalid usage of boolean expression", 0, "22000"},
	Isc_sysf_argscant_both_be_zero:        {"Arguments for @1 cannot both be zero", 1, "42000"},
	Isc_spb_no_id:                         {"missing service ID in spb", 0, "HY000"},
	Isc_ee_blr_mismatch_null:              {"External BLR message mismatch: invalid null descriptor at field @1", 1, "42000"},
	Isc_ee_blr_mismatch_length:            {"External BLR message mismatch: length = @1, expected @2", 2, "42000"},
	Isc_ss_out_of_bounds:                  {"Subscript @1 out of bounds [@2, @3]", 3, "42000"},
	Isc_missing_data_structures:           {"Install incomplete. To complete security database initialization please CREATE USER. For details read doc/README.security_database.txt.", 0, "28000"},
	Isc_protect_sys_tab:                   {"@1 operation is not allowed for system table @2", 2, "42000"},
	Isc_libtommath_generic:                {"Libtommath error code @1 in function @2", 2, "2F000"},
	Isc_wroblrver2:                        {"unsupported BLR version (expected between @1 and @2, encountered @3)", 3, "HY000"},
	Isc_trunc_limits:                      {"expected length @1, actual @2", 2, "28000"},
	Isc_info_access:                       {"Wrong info requested in isc_svc_query() for anonymous service", 0, "28000"},
	Isc_svc_no_stdin:                      {"No isc_info_svc_stdin in user request, but service thread requested stdin data", 0, "HY024"},
	Isc_svc_start_failed:                  {"Start request for anonymous service is impossible", 0, "28000"},
	Isc_svc_no_switches:                   {"All services except for getting server log require switches", 0, "HY024"},
	Isc_svc_bad_size:                      {"Size of stdin data is more than was requested from client", 0, "HY109"},
	Isc_no_crypt_plugin:                   {"Crypt plugin @1 failed to load", 1, "HY024"},
	Isc_cp_name_too_long:                  {"Length of crypt plugin name should not exceed @1 bytes", 1, "22001"},
	Isc_cp_process_active:                 {"Crypt failed - already crypting database", 0, "42818"},
	Isc_cp_already_crypted:                {"Crypt failed - database is already in requested state", 0, "42818"},
	Isc_decrypt_error:                     {"Missing crypt plugin, but page appears encrypted", 0, "XX000"},
	Isc_no_providers:                      {"No providers loaded", 0, "39000"},
	Isc_null_spb:                          {"NULL data with non-zero SPB length", 0, "42818"},
	Isc_max_args_exceeded:                 {"Maximum (@1) number of arguments exceeded for function @2", 2, "42000"},
	Isc_ee_blr_mismatch_names_count:       {"External BLR message mismatch: names count = @1, blr count = @2", 2, "42000"},
	Isc_ee_blr_mismatch_name_not_found:    {"External BLR message mismatch: name @1 not found", 1, "42000"},
	Isc_bad_result_set:                    {"Invalid resultset interface", 0, "26000"},
	Isc_wrong_message_length:              {"Message length passed from user application does not match set of columns", 0, "07000"},
	Isc_no_output_format:                  {"Resultset is missing output format information", 0, "07000"},
	Isc_item_finish:                       {"Message metadata not ready - item @1 is not finished", 1, "HY021"},
	Isc_miss_config:                       {"Missing configuration file: @1", 1, "XX000"},
	Isc_conf_line:                         {"@1: illegal line <@2>", 2, "XX000"},
	Isc_conf_include:                      {"Invalid include operator in @1 for <@2>", 2, "XX000"},
	Isc_include_depth:                     {"Include depth too big", 0, "XX000"},
	Isc_include_miss:                      {"File to include not found", 0, "XX000"},
	Isc_protect_ownership:                 {"Only the owner can change the ownership", 0, "28000"},
	Isc_badvarnum:                         {"undefined variable number", 0, "HY000"},
	Isc_sec_context:                       {"Missing security context for @1", 1, "28000"},
	Isc_multi_segment:                     {"Missing segment @1 in multisegment connect block parameter", 1, "28000"},
	Isc_login_changed:                     {"Different logins in connect and attach packets - client library error", 0, "28000"},
	Isc_auth_handshake_limit:              {"Exceeded exchange limit during authentication handshake", 0, "28000"},
	Isc_wirecrypt_incompatible:            {"Incompatible wire encryption levels requested on client and server", 0, "28000"},
	Isc_miss_wirecrypt:                    {"Client attempted to attach unencrypted but wire encryption is required", 0, "28000"},
	Isc_wirecrypt_key:                     {"Client attempted to start wire encryption using unknown key @1", 1, "28000"},
	Isc_wirecrypt_plugin:                  {"Client attempted to start wire encryption using unsupported plugin @1", 1, "28000"},
	Isc_secdb_name:                        {"Error getting security database name from configuration file", 0, "28000"},
	Isc_auth_data:                         {"Client authentication plugin is missing required data from server", 0, "28000"},
	Isc_auth_datalength:                   {"Client authentication plugin expected @2 bytes of @3 from server, got @1", 3, "28000"},
	Isc_info_unprepared_stmt:              {"Attempt to get information about an unprepared dynamic SQL statement.", 0, "HY007"},
	Isc_idx_key_value:                     {"Problematic key value is @1", 1, "HY007"},
	Isc_forupdate_virtualtbl:              {"Cannot select virtual table @1 for update WITH LOCK", 1, "HY000"},
	Isc_forupdate_systbl:                  {"Cannot select system table @1 for update WITH LOCK", 1, "HY000"},
	Isc_forupdate_temptbl:                 {"Cannot select temporary table @1 for update WITH LOCK", 1, "HY000"},
	Isc_cant_modify_sysobj:                {"System @1 @2 cannot be modified", 2, "42000"},
	Isc_server_misconfigured:              {"Server misconfigured - contact administrator please", 0, "08004"},
	Isc_alter_role:                        {"Deprecated backward compatibility ALTER ROLE ... SET/DROP AUTO ADMIN mapping may be used only for RDB$ADMIN role", 0, "0A000"},
	Isc_map_already_exists:                {"Mapping @1 already exists", 1, "42S01"},
	Isc_map_not_exists:                    {"Mapping @1 does not exist", 1, "42S02"},
	Isc_map_load:                          {"@1 failed when loading mapping cache", 1, "08004"},
	Isc_map_aster:                         {"Invalid name <*> in authentication block", 0, "08004"},
	Isc_map_multi:                         {"Multiple maps found for @1", 1, "08004"},
	Isc_map_undefined:                     {"Undefined mapping result - more than one different results found", 0, "08004"},
	Isc_baddpb_damaged_mode:               {"Incompatible mode of attachment to damaged database", 0, "HY000"},
	Isc_baddpb_buffers_range:              {"Attempt to set in database number of buffers which is out of acceptable range [@1:@2]", 2, "HY000"},
	Isc_baddpb_temp_buffers:               {"Attempt to temporarily set number of buffers less than @1", 1, "HY000"},
	Isc_map_nodb:                          {"Global mapping is not available when database @1 is not present", 1, "0A000"},
	Isc_map_notable:                       {"Global mapping is not available when table RDB$MAP is not present in database @1", 1, "0A000"},
	Isc_miss_trusted_role:                 {"Your attachment has no trusted role", 0, "0P000"},
	Isc_set_invalid_role:                  {"Role @1 is invalid or unavailable", 1, "0P000"},
	Isc_cursor_not_positioned:             {"Cursor @1 is not positioned in a valid record", 1, "HY109"},
	Isc_dup_attribute:                     {"Duplicated user attribute @1", 1, "42702"},
	Isc_dyn_no_priv:                       {"There is no privilege for this operation", 0, "42000"},
	Isc_dsql_cant_grant_option:            {"Using GRANT OPTION on @1 not allowed", 1, "42000"},
	Isc_read_conflict:                     {"read conflicts with concurrent update", 0, "40001"},
	Isc_crdb_load:                         {"@1 failed when working with CREATE DATABASE grants", 1, "08004"},
	Isc_crdb_nodb:                         {"CREATE DATABASE grants check is not possible when database @1 is not present", 1, "0A000"},
	Isc_crdb_notable:                      {"CREATE DATABASE grants check is not possible when table RDB$DB_CREATORS is not present in database @1", 1, "0A000"},
	Isc_interface_version_too_old:         {"Interface @3 version too old: expected @1, found @2", 3, "HY000"},
	Isc_fun_param_mismatch:                {"Parameter mismatch for function @1", 1, "07001"},
	Isc_savepoint_backout_err:             {"Error during savepoint backout - transaction invalidated", 0, "HY000"},
	Isc_domain_primary_key_notnull:        {"Domain used in the PRIMARY KEY constraint of table @1 must be NOT NULL", 1, "42000"},
	Isc_invalid_attachment_charset:        {"CHARACTER SET @1 cannot be used as a attachment character set", 1, "2C000"},
	Isc_map_down:                          {"Some database(s) were shutdown when trying to read mapping data", 0, "08004"},
	Isc_login_error:                       {"Error occurred during login, please check server rdbserver.log for details", 0, "08006"},
	Isc_already_opened:                    {"Database already opened with engine instance, incompatible with current", 0, "08006"},
	Isc_bad_crypt_key:                     {"Invalid crypt key @1", 1, "08006"},
	Isc_encrypt_error:                     {"Page requires encryption but crypt plugin is missing", 0, "XX000"},
	Isc_max_idx_depth:                     {"Maximum index depth (@1 levels) is reached", 1, "54000"},
	Isc_wrong_prvlg:                       {"System privilege @1 does not exist", 1, "0A000"},
	Isc_miss_prvlg:                        {"System privilege @1 is missing", 1, "28000"},
	Isc_crypt_checksum:                    {"Invalid or missing checksum of encrypted database", 0, "XX000"},
	Isc_not_dba:                           {"You must have SYSDBA rights at this server", 0, "28000"},
	Isc_no_cursor:                         {"Cannot open cursor for non-SELECT statement", 0, "07005"},
	Isc_dsql_window_incompat_frames:       {"If <window frame bound 1> specifies @1, then <window frame bound 2> shall not specify @2", 2, "42000"},
	Isc_dsql_window_range_multi_key:       {"RANGE based window with <expr> {PRECEDING | FOLLOWING} cannot have ORDER BY with more than one value", 0, "42000"},
	Isc_dsql_window_range_inv_key_type:    {"RANGE based window with <offset> PRECEDING/FOLLOWING must have a single ORDER BY key of numerical, date, time or timestamp types", 0, "42000"},
	Isc_dsql_window_frame_value_inv_type:  {"Window RANGE/ROWS/GROUPS PRECEDING/FOLLOWING value must be of a numerical type", 0, "42000"},
	Isc_window_frame_value_invalid:        {"Invalid PRECEDING or FOLLOWING offset in window function: cannot be negative", 0, "42000"},
	Isc_dsql_window_not_found:             {"Window @1 not found", 1, "42000"},
	Isc_dsql_window_cant_overr_part:       {"Cannot use PARTITION BY clause while overriding the window @1", 1, "42000"},
	Isc_dsql_window_cant_overr_order:      {"Cannot use ORDER BY clause while overriding the window @1 which already has an ORDER BY clause", 1, "42000"},
	Isc_dsql_window_cant_overr_frame:      {"Cannot override the window @1 because it has a frame clause. Tip: it can be used without parenthesis in OVER", 1, "42000"},
	Isc_dsql_window_duplicate:             {"Duplicate window definition for @1", 1, "42000"},
	Isc_sql_too_long:                      {"SQL statement is too long. Maximum size is @1 bytes.", 1, "54001"},
	Isc_cfg_stmt_timeout:                  {"Config level timeout expired.", 0, "HY008"},
	Isc_att_stmt_timeout:                  {"Attachment level timeout expired.", 0, "HY008"},
	Isc_req_stmt_timeout:                  {"Statement level timeout expired.", 0, "HY008"},
	Isc_att_shut_killed:                   {"Killed by database administrator.", 0, "08003"},
	Isc_att_shut_idle:                     {"Idle timeout expired.", 0, "08003"},
	Isc_att_shut_db_down:                  {"Database is shutdown.", 0, "08003"},
	Isc_att_shut_engine:                   {"Engine is shutdown.", 0, "08003"},
	Isc_overriding_without_identity:       {"OVERRIDING clause can be used only when an identity column is present in the INSERT's field list for table/view @1", 1, "42000"},
	Isc_overriding_system_invalid:         {"OVERRIDING SYSTEM VALUE can be used only for identity column defined as 'GENERATED ALWAYS' in INSERT for table/view @1", 1, "42000"},
	Isc_overriding_user_invalid:           {"OVERRIDING USER VALUE can be used only for identity column defined as 'GENERATED BY DEFAULT' in INSERT for table/view @1", 1, "42000"},
	Isc_overriding_missing:                {"OVERRIDING clause should be used when an identity column defined as 'GENERATED ALWAYS' is present in the INSERT's field list for table table/view @1", 1, "42000"},
	Isc_decprecision_err:                  {"DecFloat precision must be 16 or 34", 0, "HY104"},
	Isc_decfloat_divide_by_zero:           {"Decimal float divide by zero.  The code attempted to divide a DECFLOAT value by zero.", 0, "22012"},
	Isc_decfloat_inexact_result:           {"Decimal float inexact result.  The result of an operation cannot be represented as a decimal fraction.", 0, "22000"},
	Isc_decfloat_invalid_operation:        {"Decimal float invalid operation.  An indeterminant error occurred during an operation.", 0, "22000"},
	Isc_decfloat_overflow:                 {"Decimal float overflow.  The exponent of a result is greater than the magnitude allowed.", 0, "22003"},
	Isc_decfloat_underflow:                {"Decimal float underflow.  The exponent of a result is less than the magnitude allowed.", 0, "22003"},
	Isc_subfunc_notdef:                    {"Sub-function @1 has not been defined", 1, "42000"},
	Isc_subproc_notdef:                    {"Sub-procedure @1 has not been defined", 1, "42000"},
	Isc_subfunc_signat:                    {"Sub-function @1 has a signature mismatch with its forward declaration", 1, "42000"},
	Isc_subproc_signat:                    {"Sub-procedure @1 has a signature mismatch with its forward declaration", 1, "42000"},
	Isc_subfunc_defvaldecl:                {"Default values for parameters are not allowed in definition of the previously declared sub-function @1", 1, "42000"},
	Isc_subproc_defvaldecl:                {"Default values for parameters are not allowed in definition of the previously declared sub-procedure @1", 1, "42000"},
	Isc_subfunc_not_impl:                  {"Sub-function @1 was declared but not implemented", 1, "42000"},
	Isc_subproc_not_impl:                  {"Sub-procedure @1 was declared but not implemented", 1, "42000"},
	Isc_sysf_invalid_hash_algorithm:       {"Invalid HASH algorithm @1", 1, "42000"},
	Isc_expression_eval_index:             {"Expression evaluation error for index @1 on table @2", 2, "42000"},
	Isc_invalid_decfloat_trap:             {"Invalid decfloat trap state @1", 1, "42000"},
	Isc_invalid_decfloat_round:            {"Invalid decfloat rounding mode @1", 1, "42000"},
	Isc_sysf_invalid_first_last_part:      {"Invalid part @1 to calculate the @1 of a DATE/TIMESTAMP", 2, "42000"},
	Isc_sysf_invalid_date_timestamp:       {"Expected DATE/TIMESTAMP value in @1", 1, "42000"},
	Isc_precision_err2:                    {"Precision must be from @1 to @2", 2, "HY104"},
	Isc_bad_batch_handle:                  {"invalid batch handle", 0, "HY000"},
	Isc_intl_char:                         {"Bad international character in tag @1", 1, "22021"},
	Isc_null_block:                        {"Null data in parameters block with non-zero length", 0, "22000"},
	Isc_mixed_info:                        {"Items working with running service and getting generic server information should not be mixed in single info block", 0, "22000"},
	Isc_unknown_info:                      {"Unknown information item, code @1", 1, "22000"},
	Isc_bpb_version:                       {"Wrong version of blob parameters block @1, should be @2", 2, "22000"},
	Isc_user_manager:                      {"User management plugin is missing or failed to load", 0, "28000"},
	Isc_icu_entrypoint:                    {"Missing entrypoint @1 in ICU library", 1, "XX000"},
	Isc_icu_library:                       {"Could not find acceptable ICU library", 0, "XX000"},
	Isc_metadata_name:                     {"Name @1 not found in system MetadataBuilder", 1, "02000"},
	Isc_tokens_parse:                      {"Parse to tokens error", 0, "42000"},
	Isc_iconv_open:                        {"Error opening international conversion descriptor from @1 to @2", 2, "08006"},
	Isc_batch_compl_range:                 {"Message @1 is out of range, only @2 messages in batch", 2, "07002"},
	Isc_batch_compl_detail:                {"Detailed error info for message @1 is missing in batch", 1, "07002"},
	Isc_deflate_init:                      {"Compression stream init error @1", 1, "08006"},
	Isc_inflate_init:                      {"Decompression stream init error @1", 1, "08006"},
	Isc_big_segment:                       {"Segment size (@1) should not exceed 65535 (64K - 1) when using segmented blob", 1, "22026"},
	Isc_batch_policy:                      {"Invalid blob policy in the batch for @1() call", 1, "22000"},
	Isc_batch_defbpb:                      {"Can't change default BPB after adding any data to batch", 0, "22000"},
	Isc_batch_align:                       {"Unexpected info buffer structure querying for server batch parameters", 0, "22000"},
	Isc_multi_segment_dup:                 {"Duplicated segment @1 in multisegment connect block parameter", 1, "22000"},
	Isc_non_plugin_protocol:               {"Plugin not supported by network protocol", 0, "08004"},
	Isc_message_format:                    {"Error parsing message format", 0, "22000"},
	Isc_batch_param_version:               {"Wrong version of batch parameters block @1, should be @2", 2, "22000"},
	Isc_batch_msg_long:                    {"Message size (@1) in batch exceeds internal buffer size (@2)", 2, "54000"},
	Isc_batch_open:                        {"Batch already opened for this statement", 0, "HY010"},
	Isc_batch_type:                        {"Invalid type of statement used in batch", 0, "07003"},
	Isc_batch_param:                       {"Statement used in batch must have parameters", 0, "07001"},
	Isc_batch_blobs:                       {"There are no blobs in associated with batch statement", 0, "07001"},
	Isc_batch_blob_append:                 {"appendBlobData() is used to append data to last blob but no such blob was added to the batch", 0, "22000"},
	Isc_batch_stream_align:                {"Portions of data, passed as blob stream, should have size multiple to the alignment required for blobs", 0, "22000"},
	Isc_batch_rpt_blob:                    {"Repeated blob id @1 in registerBlob()", 1, "22000"},
	Isc_batch_blob_buf:                    {"Blob buffer format error", 0, "22000"},
	Isc_batch_small_data:                  {"Unusable (too small) data remained in @1 buffer", 1, "22000"},
	Isc_batch_cont_bpb:                    {"Blob continuation should not contain BPB", 0, "22000"},
	Isc_batch_big_bpb:                     {"Size of BPB (@1) greater than remaining data (@2)", 2, "22000"},
	Isc_batch_big_segment:                 {"Size of segment (@1) greater than current BLOB data (@2)", 2, "22000"},
	Isc_batch_big_seg2:                    {"Size of segment (@1) greater than available data (@2)", 2, "22000"},
	Isc_batch_blob_id:                     {"Unknown blob ID @1 in the batch message", 1, "22000"},
	Isc_batch_too_big:                     {"Internal buffer overflow - batch too big", 0, "54000"},
	Isc_num_literal:                       {"Numeric literal too long", 0, "54000"},
	Isc_map_event:                         {"Error using events in mapping shared memory: @1", 1, "XX000"},
	Isc_map_overflow:                      {"Global mapping memory overflow", 0, "54000"},
	Isc_hdr_overflow:                      {"Header page overflow - too many clumplets on it", 0, "54000"},
	Isc_vld_plugins:                       {"No matching client/server authentication plugins configured for execute statement in embedded datasource", 0, "28000"},
	Isc_db_crypt_key:                      {"Missing database encryption key for your attachment", 0, "08004"},
	Isc_no_keyholder_plugin:               {"Key holder plugin @1 failed to load", 1, "HY024"},
	Isc_ses_reset_err:                     {"Cannot reset user session", 0, "01002"},
	Isc_ses_reset_open_trans:              {"There are open transactions (@1 active)", 1, "25S02"},
	Isc_ses_reset_warn:                    {"Session was reset with warning(s)", 0, "01000"},
	Isc_ses_reset_tran_rollback:           {"Transaction is rolled back due to session reset, all changes are lost", 0, "01102"},
	Isc_plugin_name:                       {"Plugin @1:", 1, "00000"},
	Isc_parameter_name:                    {"PARAMETER @1", 1, "42000"},
	Isc_file_starting_page_err:            {"Starting page number for file @1 must be @2 or greater", 2, "HY000"},
	Isc_invalid_timezone_offset:           {"Invalid time zone offset: @1 - must use format +/-hours:minutes and be between -14:00 and +14:00", 1, "22009"},
	Isc_invalid_timezone_region:           {"Invalid time zone region: @1", 1, "22009"},
	Isc_invalid_timezone_id:               {"Invalid time zone ID: @1", 1, "22009"},
	Isc_tom_decode64len:                   {"Wrong base64 text length @1, should be multiple of 4", 1, "22023"},
	Isc_tom_strblob:                       {"Invalid first parameter datatype - need string or blob", 0, "22023"},
	Isc_tom_reg:                           {"Error registering @1 - probably bad tomcrypt library", 1, "22023"},
	Isc_tom_algorithm:                     {"Unknown crypt algorithm @1 in USING clause", 1, "22023"},
	Isc_tom_mode_miss:                     {"Should specify mode parameter for symmetric cipher", 0, "22023"},
	Isc_tom_mode_bad:                      {"Unknown symmetric crypt mode specified", 0, "22023"},
	Isc_tom_no_mode:                       {"Mode parameter makes no sense for chosen cipher", 0, "22023"},
	Isc_tom_iv_miss:                       {"Should specify initialization vector (IV) for chosen cipher and/or mode", 0, "22023"},
	Isc_tom_no_iv:                         {"Initialization vector (IV) makes no sense for chosen cipher and/or mode", 0, "22023"},
	Isc_tom_ctrtype_bad:                   {"Invalid counter endianess @1", 1, "22023"},
	Isc_tom_no_ctrtype:                    {"Counter endianess parameter is not used in mode @1", 1, "22023"},
	Isc_tom_ctr_big:                       {"Too big counter value @1, maximum @2 can be used", 2, "22023"},
	Isc_tom_no_ctr:                        {"Counter length/value parameter is not used with @1 @2", 2, "22023"},
	Isc_tom_iv_length:                     {"Invalid initialization vector (IV) length @1, need @2", 2, "22023"},
	Isc_tom_error:                         {"TomCrypt library error: @1", 1, "22000"},
	Isc_tom_yarrow_start:                  {"Starting PRNG yarrow", 0, "22023"},
	Isc_tom_yarrow_setup:                  {"Setting up PRNG yarrow", 0, "22023"},
	Isc_tom_init_mode:                     {"Initializing @1 mode", 1, "22023"},
	Isc_tom_crypt_mode:                    {"Encrypting in @1 mode", 1, "22023"},
	Isc_tom_decrypt_mode:                  {"Decrypting in @1 mode", 1, "22023"},
	Isc_tom_init_cip:                      {"Initializing cipher @1", 1, "22023"},
	Isc_tom_crypt_cip:                     {"Encrypting using cipher @1", 1, "22023"},
	Isc_tom_decrypt_cip:                   {"Decrypting using cipher @1", 1, "22023"},
	Isc_tom_setup_cip:                     {"Setting initialization vector (IV) for @1", 1, "22023"},
	Isc_tom_setup_chacha:                  {"Invalid initialization vector (IV) length @1, need  8 or 12", 1, "22023"},
	Isc_tom_encode:                        {"Encoding @1", 1, "22023"},
	Isc_tom_decode:                        {"Decoding @1", 1, "22023"},
	Isc_tom_rsa_import:                    {"Importing RSA key", 0, "22023"},
	Isc_tom_oaep:                          {"Invalid OAEP packet", 0, "22023"},
	Isc_tom_hash_bad:                      {"Unknown hash algorithm @1", 1, "22023"},
	Isc_tom_rsa_make:                      {"Making RSA key", 0, "22023"},
	Isc_tom_rsa_export:                    {"Exporting @1 RSA key", 1, "22023"},
	Isc_tom_rsa_sign:                      {"RSA-signing data", 0, "22023"},
	Isc_tom_rsa_verify:                    {"Verifying RSA-signed data", 0, "22023"},
	Isc_tom_chacha_key:                    {"Invalid key length @1, need 16 or 32", 1, "22023"},
	Isc_bad_repl_handle:                   {"invalid replicator handle", 0, "08003"},
	Isc_tra_snapshot_does_not_exist:       {"Transaction's base snapshot number does not exist", 0, "0B000"},
	Isc_eds_input_prm_not_used:            {"Input parameter '@1' is not used in SQL query text", 1, "42000"},
	Isc_effective_user:                    {"Effective user is @1", 1, "28000"},
	Isc_invalid_time_zone_bind:            {"Invalid time zone bind mode @1", 1, "42000"},
	Isc_invalid_decfloat_bind:             {"Invalid decfloat bind mode @1", 1, "42000"},
	Isc_odd_hex_len:                       {"Invalid hex text length @1, should be multiple of 2", 1, "22023"},
	Isc_invalid_hex_digit:                 {"Invalid hex digit @1 at position @2", 2, "22023"},
	Isc_bind_err:                          {"Error processing isc_dpb_set_bind clumplet \"@1\"", 1, "08004"},
	Isc_bind_statement:                    {"The following statement failed: @1", 1, "08004"},
	Isc_bind_convert:                      {"Can not convert @1 to @2", 2, "22000"},
	Isc_cannot_update_old_blob:            {"cannot update old BLOB", 0, "42000"},
	Isc_cannot_read_new_blob:              {"cannot read from new BLOB", 0, "42000"},
	Isc_dyn_no_create_priv:                {"No permission for CREATE @1 operation", 1, "42000"},
	Isc_suspend_without_returns:           {"SUSPEND could not be used without RETURNS clause in PROCEDURE or EXECUTE BLOCK", 0, "42000"},
	Isc_truncate_warn:                     {"String truncated warning due to the following reason", 0, "01004"},
	Isc_truncate_monitor:                  {"Monitoring data does not fit into the field", 0, "01004"},
	Isc_truncate_context:                  {"Engine data does not fit into return value of system function", 0, "01004"},
	Isc_merge_dup_update:                  {"Multiple source records cannot match the same target during MERGE", 0, "21000"},
	Isc_wrong_page:                        {"RDB$PAGES written by non-system transaction, DB appears to be damaged", 0, "XX000"},
	Isc_repl_error:                        {"Replication error", 0, "HY000"},
	Isc_ses_reset_failed:                  {"Reset of user session failed. Connection is shut down.", 0, "08003"},
	Isc_block_size:                        {"File size is less than expected", 0, "XX000"},
	Isc_tom_key_length:                    {"Invalid key length @1, need >@2", 2, "22023"},
	Isc_inf_invalid_args:                  {"Invalid information arguments", 0, "HY000"},
	Isc_sysf_invalid_null_empty:           {"Empty or NULL parameter @1 is not accepted", 1, "22023"},
	Isc_bad_loctab_num:                    {"Undefined local table number @1", 1, "HY000"},
	Isc_quoted_str_bad:                    {"Invalid text <@1> after quoted string", 1, "22024"},
	Isc_quoted_str_miss:                   {"Missing terminating quote <@1> in the end of quoted string", 1, "22024"},
	Isc_wrong_shmem_ver:                   {"@1: inconsistent shared memory type/version; found @2, expected @3", 3, "08006"},
	Isc_wrong_shmem_bitness:               {"@1-bit engine can't open database already opened by @2-bit engine", 2, "08006"},
	Isc_wrong_proc_plan:                   {"Procedures cannot specify access type other than NATURAL in the plan", 0, "HY000"},
	Isc_invalid_blob_util_handle:          {"Invalid RDB$BLOB_UTIL handle", 0, "42000"},
	Isc_bad_temp_blob_id:                  {"Invalid temporary BLOB ID", 0, "42000"},
	Isc_ods_upgrade_err:                   {"ODS upgrade failed while adding new system %s", 0, "HY000"},
	Isc_bad_par_workers:                   {"Wrong parallel workers value @1, valid range is from 1 to @2", 2, "HY000"},
	Isc_idx_expr_not_found:                {"Definition of index expression is not found for index @1", 1, "42000"},
	Isc_idx_cond_not_found:                {"Definition of index condition is not found for index @1", 1, "42000"},
	Isc_uninitialized_var:                 {"Variable @1 is not initialized", 1, "42000"},
	Isc_param_not_exist:                   {"Parameter @1 does not exist", 1, "07001"},
	Isc_param_no_default_not_specified:    {"Parameter @1 has no default value and was not specified or was specified with DEFAULT", 1, "07001"},
	Isc_param_multiple_assignments:        {"Parameter @1 has multiple assignments", 1, "07001"},
	Isc_invalid_date_format:               {"Cannot recognize \"@1\" part of date format", 1, "HY000"},
	Isc_invalid_raw_string_in_date_format: {"Cannot find closing \" for raw text in date format", 0, "HY000"},
	Isc_invalid_data_type_for_date_format: {"It is not possible to use this data type for date formatting", 0, "HY000"},
	Isc_incompatible_date_format_with_current_date_type:           {"Cannot use \"@1\" format with current date type", 1, "HY000"},
	Isc_value_for_pattern_is_out_of_range:                         {"Value for @1 pattern is out of range [@2, @3]", 3, "HY000"},
	Isc_month_name_mismatch:                                       {"@1 is not MONTH", 1, "HY000"},
	Isc_incorrect_hours_period:                                    {"@1 is incorrect period for 12H, it should be A.M. or P.M.", 1, "HY000"},
	Isc_data_for_format_is_exhausted:                              {"All data has been read, but format pattern wants more. Unfilled patterns: \"@1\"", 1, "HY000"},
	Isc_trailing_part_of_string:                                   {"There is a trailing part of input string that does not fit into FORMAT: \"@1\"", 1, "HY000"},
	Isc_pattern_cant_be_used_without_other_pattern:                {"@1 can't be used without @2", 2, "HY000"},
	Isc_pattern_cant_be_used_without_other_pattern_and_vice_versa: {"@1 can't be used without @2 and vice versa", 2, "HY000"},
	Isc_incompatible_format_patterns:                              {"@1 incompatible with @2", 2, "HY000"},
	Isc_only_one_pattern_can_be_used:                              {"Can use only one of these patterns @1", 1, "HY000"},
	Isc_can_not_use_same_pattern_twice:                            {"Cannot use the same pattern twice: @1", 1, "HY000"},
	Isc_sysf_invalid_gen_uuid_version:                             {"Invalid GEN_UUID version (@1). Must be 4 or 7", 1, "42000"},
	Isc_sweep_unable_to_run:                                       {"Unable to run sweep", 0, "42000"},
	Isc_sweep_concurrent_instance:                                 {"Another instance of sweep is already running", 0, "42000"},
	Isc_sweep_read_only:                                           {"Database in read only state", 0, "42000"},
	Isc_sweep_attach_no_cleanup:                                   {"Attachment has no cleanup flag set", 0, "42000"},
	Isc_invalid_timezone_region_or_displacement:                   {"Invalid time zone region or displacement: @1", 1, "HY000"},
	Isc_argmustbe_exact_range_for:                                 {"Arguments for range-based FOR must be exact numeric types", 0, "42000"},
	Isc_range_for_by_should_be_positive:                           {"Range-based FOR BY argument must be positive", 0, "42000"},
	Isc_missing_value_for_format_pattern:                          {"Cannot find value in input string for \"@1\" pattern", 1, "HY000"},
	Isc_invalid_name:                                              {"Invalid name: @1", 1, "HY000"},
	Isc_invalid_unqualified_name_list:                             {"Invalid list of unqualified names: @1", 1, "HY000"},
	Isc_no_user_att_while_restore:                                 {"User attachments are not allowed for the database being restored", 0, "HY000"},
	Isc_genseq_stepmustbe_nonzero:                                 {"Argument STEP must be different than zero for function @1", 1, "42000"},
	Isc_argmustbe_exact_function:                                  {"Arguments for @1 function must be exact numeric types", 1, "42000"},
	Isc_sysf_argmustbe_range_inc0_1:                               {"Argument for @1 must be in the range [0, 1]", 1, "42000"},
	Isc_argmustbe_numeric_function:                                {"Argument for @1 function must be numeric types", 1, "42000"},
	Isc_percetile_only_one_sort_item:                              {"The PERCENTILE_DISC and PERENTILE_CONT functions support only one sort item in WITHIN GROUP", 0, "42000"},
	Isc_argmustbe_const_within_group:                              {"Argument for @1 function must be constant within each group", 1, "42000"},
	Isc_update_overwrite:                                          {"UPDATE will overwrite changes made by the trigger or by the another UPDATE in the same cursor", 0, "27000"},
	Isc_pman_plugin_dirname:                                       {"Plugin @1 can not be loaded - name should not contain directory separator and path component", 1, "00000"},
	Isc_private_constant:                                          {"The constant @1 is private", 1, "42000"},
	Isc_package_alias_help:                                        {"Use an alias to resolve the conflict", 0, "42000"},
	Isc_bad_constant_blr:                                          {"Error while parsing BLR value of the constant @1", 1, "2F000"},
	Isc_bad_constant_desc:                                         {"Error while reading type of the constant with name @1", 1, "2F000"},
	Isc_bad_constant_name:                                         {"Constant @1 cannot be found", 1, "2F000"},
	Isc_bad_constant_type:                                         {"@1 is not supported to be a constant type", 1, "2F000"},
	Isc_not_defined_constant:                                      {"The constant @1 is not defined", 1, "42000"},
	Isc_const_name:                                                {"CONSTANT @1", 1, "42000"},
	Isc_private_table:                                             {"Table @1 is private to package @2", 2, "42000"},
	Isc_temp_space_invalid_pos:                                    {"Invalid position to read/write in a temporary file (positon: @1, size: @2)", 2, "HY000"},
	Isc_dsql_agg_non_agg_context:                                  {"Aggregate function @1 cannot be used in non-aggregate context", 1, "42000"},
	Isc_dsql_agg_param_not_accum:                                  {"Aggregate function input parameters may be referenced only in ON ACCUMULATE DO", 0, "42000"},
	Isc_dsql_agg_exit_group:                                       {"EXIT is not allowed in ON GROUP DO section of aggregate function", 0, "42000"},
	Isc_dsql_agg_return:                                           {"RETURN is not allowed in ON START DO, ON ACCUMULATE DO or ON FINISH DO sections of aggregate function; use EXIT instead", 0, "42000"},
	Isc_bad_ext_file:                                              {"Invalid external file format", 0, "22000"},
	Isc_wrong_adp_fields_def:                                      {"Wrong set of adapter fields", 0, "42000"},
	Isc_wrong_adp_field_name:                                      {"Adapter field named \"@1\" is not supported", 1, "42000"},
	Isc_wrong_adp_field_type:                                      {"Wrong type of adapter field \"@1\"", 1, "42000"},
	Isc_bad_ext_record:                                            {"Unknown record type in external file \"@1\" at offset @2", 2, "22000"},
	Isc_bad_adp_type:                                              {"Unknown adapter type \"@1\"", 1, "42000"},
	Isc_dir_resolve_error:                                         {"Cannot resolve directory name \"@1\"", 1, "HY000"},
	Isc_read_file_error:                                           {"Error reading file \"@1\": @2", 2, "HY000"},
	Isc_create_file_error:                                         {"Error creating file \"@1\": @2", 2, "HY000"},
	Isc_delete_file_error:                                         {"Error deleting file \"@1\": @2", 2, "HY000"},
	Isc_blob_io_error:                                             {"I/O error during \"@1\" operation for BLOB file \"@2\"", 2, "42000"},
	Isc_cant_connect_to_ldap:                                      {"Cannot connect to LDAP server: @1", 1, "28000"},
	Isc_cant_bind_to_ldap:                                         {"Cannot bind to LDAP server with specified login and password", 0, "28000"},
	Isc_no_ldap_init:                                              {"Cannot initialize LDAP plugin", 0, "28000"},
	Isc_sysf_ldap_attr:                                            {"Cannot get user attribute \"@1\" from LDAP", 1, "42000"},
	Isc_no_ext_file:                                               {"Missing external file for adapter", 0, "22000"},
	Isc_no_gss_init:                                               {"GSS library was not loaded correctly", 0, "HY000"},
	Isc_hash_error:                                                {"Hash calculation error", 0, "HY000"},
	Isc_policy_already_exists:                                     {"Policy @1 already exists", 1, "22028"},
	Isc_policy_not_exists:                                         {"Policy @1 does not exist", 1, "02000"},
	Isc_user_not_exists:                                           {"User @1 does not exist", 1, "02000"},
	Isc_policy_is_granted:                                         {"Policy @1 is granted to a user", 1, "23000"},
	Isc_bad_trig_BLR:                                              {"Error while parsing trigger @1's BLR", 1, "2F000"},
	Isc_not_rdb:                                                   {"Unable to open database @1, is not Red Database", 1, "HY000"},
	335546344:                                                     {"Effective user is @1", 1, ""},
	Isc_trace_write_error:                                         {"Error when trying to write to the trace file", 0, "22000"},
	Isc_no_granted_policy:                                         {"User @1 does not have granted policy", 1, "02000"},
	Isc_regexp_substr_error:                                       {"Error calculating substring using regular expression: @2", 1, "42000"},
	Isc_secdb_coonection_error:                                    {"Error while trying to attach the security DB", 0, "22000"},
	Isc_policies_reading_error:                                    {"Error while trying to read the PLG$POLICIES", 0, "22000"},
	Isc_get_policies_error:                                        {"@1 failed during selecting from SEC$POLICIES", 1, "22000"},
	Isc_att_suspended:                                             {"Connection suspended", 0, "22000"},
	Isc_suspend_privileges_error:                                  {"not enough privileges to complete operation", 0, "22000"},
	Isc_trace_buffer_limit:                                        {"Size of trace message buffer (@1) exceeds limit @2", 2, "HY000"},
	Isc_pman_hash_incorrect:                                       {"Hash is incorrect", 0, "00000"},
	Isc_json_table_plan_missing_name:                              {"The nested column name @1 does not appear in the plan clause", 1, "22000"},
	Isc_json_table_same_names:                                     {"The column name @1 belongs to several columns", 1, "22000"},
	Isc_json_table_plan_same_names:                                {"The path name @1 belongs to several columns", 1, "22000"},
	Isc_json_table_plan_double_appearance:                         {"All the path names (including @1) must appear in the PLAN clause exactly once", 1, "22000"},
	Isc_json_table_plan_wrong_level:                               {"The nested @1 is on the wrong level ib the plan", 1, "22000"},
	Isc_json_table_invalid_plan:                                   {"The json table plan is invalid", 0, "22000"},
	Isc_json_table_plan_wrong_name:                                {"There is no nested column name with named @1", 1, "22000"},
	Isc_foreign_server:                                            {"foreign server @1", 1, "42000"},
	Isc_foreign_provider_not_found:                                {"Foreign provider '@1' for server '@2' not found", 2, "42000"},
	Isc_change_forced_write_mode_with_connections:                 {"Can't change forced write mode while server has connections", 0, "42000"},
	Isc_license_not_found:                                         {"License plugin is not found", 0, "HY000"},
	Isc_feature_not_available:                                     {"@1 is not available in this edition", 1, "HY000"},
	Isc_repl_mispos:                                               {"Replication packet is mispositioned. Replica position is (@1:@2)", 2, "HY000"},
	Isc_missing_trace_plugin:                                      {"The '@1' plugin is missing. Please specify it in the TracePlugin option and restart the server", 1, "HY000"},
	Isc_repl_recover_err:                                          {"Database recovery failed", 0, "HY000"},
	Isc_user_policy_reading_error:                                 {"Error while trying to read the PLG$USER_POLICY", 0, "22000"},
	Isc_get_user_policy_error:                                     {"@1 failed during selecting from SEC$USER_POLICY", 1, "22000"},
	Isc_gfix_db_name:                                              {"data base file name (@1) already given", 1, "00000"},
	Isc_gfix_invalid_sw:                                           {"invalid switch @1", 1, "00000"},
	Isc_gfix_version:                                              {"gfix version @1", 1, ""},
	Isc_gfix_incmp_sw:                                             {"incompatible switch combination", 0, "00000"},
	Isc_gfix_replay_req:                                           {"replay log pathname required", 0, "00000"},
	Isc_gfix_pgbuf_req:                                            {"number of page buffers for cache required", 0, "00000"},
	Isc_gfix_val_req:                                              {"numeric value required", 0, "00000"},
	Isc_gfix_pval_req:                                             {"positive numeric value required", 0, "00000"},
	Isc_gfix_trn_req:                                              {"number of transactions per sweep required", 0, "00000"},
	Isc_gfix_trn_all_req:                                          {"transaction number or \"all\" required", 0, ""},
	Isc_gfix_sync_req:                                             {"\"sync\" or \"async\" required", 0, ""},
	Isc_gfix_full_req:                                             {"\"full\" or \"reserve\" required", 0, "00000"},
	Isc_gfix_usrname_req:                                          {"user name required", 0, "00000"},
	Isc_gfix_pass_req:                                             {"password required", 0, "00000"},
	Isc_gfix_subs_name:                                            {"subsystem name", 0, "00000"},
	Isc_gfix_wal_req:                                              {"\"wal\" required", 0, "00000"},
	Isc_gfix_sec_req:                                              {"number of seconds required", 0, "00000"},
	Isc_gfix_nval_req:                                             {"numeric value between 0 and 32767 inclusive required", 0, "00000"},
	Isc_gfix_type_shut:                                            {"must specify type of shutdown", 0, "00000"},
	Isc_gfix_retry:                                                {"please retry, specifying an option", 0, "00000"},
	Isc_gfix_opt:                                                  {"plausible options are:", 0, ""},
	Isc_gfix_qualifiers:                                           {"\\n    Options can be abbreviated to the unparenthesized characters", 0, ""},
	Isc_gfix_retry_db:                                             {"please retry, giving a database name", 0, "00000"},
	Isc_gfix_summary:                                              {"Summary of validation errors", 0, ""},
	Isc_gfix_opt_active:                                           {"   -ac(tivate_shadow)   activate shadow file for database usage", 0, ""},
	Isc_gfix_opt_attach:                                           {"   -at(tach)            shutdown new database attachments", 0, ""},
	Isc_gfix_opt_begin_log:                                        {"\t-begin_log\tbegin logging for replay utility", 0, ""},
	Isc_gfix_opt_buffers:                                          {"   -b(uffers)           set page buffers <n>", 0, ""},
	Isc_gfix_opt_commit:                                           {"   -co(mmit)            commit transaction <tr / all>", 0, ""},
	Isc_gfix_opt_cache:                                            {"   -ca(che)             shutdown cache manager", 0, ""},
	Isc_gfix_opt_disable:                                          {"\t-disable\tdisable WAL", 0, ""},
	Isc_gfix_opt_full:                                             {"   -fu(ll)              validate record fragments (-v)", 0, ""},
	Isc_gfix_opt_force:                                            {"   -fo(rce_shutdown)    force database shutdown", 0, ""},
	Isc_gfix_opt_housekeep:                                        {"   -h(ousekeeping)      set sweep interval <n>", 0, ""},
	Isc_gfix_opt_ignore:                                           {"   -i(gnore)            ignore checksum errors", 0, ""},
	Isc_gfix_opt_kill:                                             {"   -k(ill_shadow)       kill all unavailable shadow files", 0, ""},
	Isc_gfix_opt_list:                                             {"   -l(ist)              show limbo transactions", 0, ""},
	Isc_gfix_opt_mend:                                             {"   -me(nd)              prepare corrupt database for backup", 0, ""},
	Isc_gfix_opt_no_update:                                        {"   -n(o_update)         read-only validation (-v)", 0, ""},
	Isc_gfix_opt_online:                                           {"   -o(nline)            database online <single / multi / normal>", 0, ""},
	Isc_gfix_opt_prompt:                                           {"   -pr(ompt)            prompt for commit/rollback (-l)", 0, ""},
	Isc_gfix_opt_password:                                         {"   -pa(ssword)          default password", 0, ""},
	Isc_gfix_opt_quit_log:                                         {"\t-quit_log\tquit logging for replay utility", 0, ""},
	Isc_gfix_opt_rollback:                                         {"   -r(ollback)          rollback transaction <tr / all>", 0, ""},
	Isc_gfix_opt_sweep:                                            {"   -sw(eep)             force garbage collection", 0, ""},
	Isc_gfix_opt_shut:                                             {"   -sh(utdown)          shutdown <full / single / multi>", 0, ""},
	Isc_gfix_opt_two_phase:                                        {"   -tw(o_phase)         perform automated two-phase recovery", 0, ""},
	Isc_gfix_opt_tran:                                             {"   -tra(nsaction)       shutdown transaction startup", 0, ""},
	Isc_gfix_opt_use:                                              {"   -u(se)               use full or reserve space for versions", 0, ""},
	Isc_gfix_opt_user:                                             {"   -user                default user name", 0, ""},
	Isc_gfix_opt_validate:                                         {"   -v(alidate)          validate database structure", 0, ""},
	Isc_gfix_opt_write:                                            {"   -w(rite)             write synchronously or asynchronously", 0, ""},
	Isc_gfix_opt_x:                                                {"   -x                   set debug on", 0, ""},
	Isc_gfix_opt_z:                                                {"   -z                   print software version number", 0, ""},
	Isc_gfix_rec_err:                                              {"\\n\tNumber of record level errors\t: @1", 1, ""},
	Isc_gfix_blob_err:                                             {"\tNumber of Blob page errors\t: @1", 1, ""},
	Isc_gfix_data_err:                                             {"\tNumber of data page errors\t: @1", 1, ""},
	Isc_gfix_index_err:                                            {"\tNumber of index page errors\t: @1", 1, ""},
	Isc_gfix_pointer_err:                                          {"\tNumber of pointer page errors\t: @1", 1, ""},
	Isc_gfix_trn_err:                                              {"\tNumber of transaction page errors\t: @1", 1, ""},
	Isc_gfix_db_err:                                               {"\tNumber of database page errors\t: @1", 1, ""},
	Isc_gfix_bad_block:                                            {"bad block type", 0, ""},
	Isc_gfix_exceed_max:                                           {"internal block exceeds maximum size", 0, "00000"},
	Isc_gfix_corrupt_pool:                                         {"corrupt pool", 0, "00000"},
	Isc_gfix_mem_exhausted:                                        {"virtual memory exhausted", 0, "00000"},
	Isc_gfix_bad_pool:                                             {"bad pool id", 0, "00000"},
	Isc_gfix_trn_not_valid:                                        {"Transaction state @1 not in valid range.", 1, "00000"},
	Isc_gfix_dbg_attach:                                           {"ATTACH_DATABASE: attempted attach of @1,", 1, ""},
	Isc_gfix_dbg_failed:                                           {" failed", 0, ""},
	Isc_gfix_dbg_success:                                          {" succeeded", 0, ""},
	Isc_gfix_trn_limbo:                                            {"Transaction @1 is in limbo.", 1, ""},
	Isc_gfix_try_again:                                            {"More limbo transactions than fit.  Try again", 0, ""},
	Isc_gfix_unrec_item:                                           {"Unrecognized info item @1", 1, ""},
	Isc_gfix_commit_violate:                                       {"A commit of transaction @1 will violate two-phase commit.", 1, ""},
	Isc_gfix_preserve:                                             {"A rollback of transaction @1 is needed to preserve two-phase commit.", 1, ""},
	Isc_gfix_part_commit:                                          {"Transaction @1 has already been partially committed.", 1, ""},
	Isc_gfix_rback_violate:                                        {"A rollback of this transaction will violate two-phase commit.", 0, ""},
	Isc_gfix_part_commit2:                                         {"Transaction @1 has been partially committed.", 1, ""},
	Isc_gfix_commit_pres:                                          {"A commit is necessary to preserve the two-phase commit.", 0, ""},
	Isc_gfix_insuff_info:                                          {"Insufficient information is available to determine", 0, ""},
	Isc_gfix_action:                                               {"a proper action for transaction @1.", 1, ""},
	Isc_gfix_all_prep:                                             {"Transaction @1: All subtransactions have been prepared.", 1, ""},
	Isc_gfix_comm_rback:                                           {"Either commit or rollback is possible.", 0, ""},
	Isc_gfix_unexp_eoi:                                            {"unexpected end of input", 0, "00000"},
	Isc_gfix_ask:                                                  {"Commit, rollback, or neither (c, r, or n)?", 0, ""},
	Isc_gfix_reattach_failed:                                      {"Could not reattach to database for transaction @1.", 1, ""},
	Isc_gfix_org_path:                                             {"Original path: @1", 1, ""},
	Isc_gfix_enter_path:                                           {"Enter a valid path:", 0, ""},
	Isc_gfix_att_unsucc:                                           {"Attach unsuccessful.", 0, ""},
	Isc_gfix_recon_fail:                                           {"failed to reconnect to a transaction in database @1", 1, "00000"},
	Isc_gfix_trn2:                                                 {"Transaction @1:", 1, ""},
	Isc_gfix_mdb_trn:                                              {"  Multidatabase transaction:", 0, ""},
	Isc_gfix_host_site:                                            {"    Host Site: @1", 1, ""},
	Isc_gfix_trn:                                                  {"    Transaction @1", 1, ""},
	Isc_gfix_prepared:                                             {"has been prepared.", 0, ""},
	Isc_gfix_committed:                                            {"has been committed.", 0, ""},
	Isc_gfix_rolled_back:                                          {"has been rolled back.", 0, ""},
	Isc_gfix_not_available:                                        {"is not available.", 0, ""},
	Isc_gfix_not_prepared:                                         {"is not found, assumed not prepared.", 0, ""},
	Isc_gfix_be_committed:                                         {"is not found, assumed to be committed.", 0, ""},
	Isc_gfix_rmt_site:                                             {"        Remote Site: @1", 1, ""},
	Isc_gfix_db_path:                                              {"        Database Path: @1", 1, ""},
	Isc_gfix_auto_comm:                                            {"  Automated recovery would commit this transaction.", 0, ""},
	Isc_gfix_auto_rback:                                           {"  Automated recovery would rollback this transaction.", 0, ""},
	Isc_gfix_warning:                                              {"Warning: Multidatabase transaction is in inconsistent state for recovery.", 0, ""},
	Isc_gfix_trn_was_comm:                                         {"Transaction @1 was committed, but prior ones were rolled back.", 1, ""},
	Isc_gfix_trn_was_rback:                                        {"Transaction @1 was rolled back, but prior ones were committed.", 1, ""},
	Isc_gfix_trn_unknown:                                          {"Transaction description item unknown", 0, "00000"},
	Isc_gfix_opt_mode:                                             {"   -mo(de)              read_only or read_write database", 0, ""},
	Isc_gfix_mode_req:                                             {"\"read_only\" or \"read_write\" required", 0, "00000"},
	Isc_gfix_opt_SQL_dialect:                                      {"   -sq(l_dialect)       set database dialect n", 0, ""},
	Isc_gfix_SQL_dialect:                                          {"database SQL dialect must be one of '@1'", 1, ""},
	Isc_gfix_dialect_req:                                          {"dialect number required", 0, ""},
	Isc_gfix_pzval_req:                                            {"positive or zero numeric value required", 0, "00000"},
	Isc_gfix_opt_trusted:                                          {"   -tru(sted)           use trusted authentication", 0, ""},
	335741044:                                                     {"could not open password file @1, errno @2", 2, ""},
	335741045:                                                     {"could not read password file @1, errno @2", 2, ""},
	335741046:                                                     {"empty password file @1", 1, ""},
	335741047:                                                     {"   -fe(tch_password)    fetch password from file", 0, ""},
	335741048:                                                     {"usage: gfix [options] <database>", 0, ""},
	Isc_gfix_opt_nolinger:                                         {"   -nol(inger)          close database ignoring linger setting for it", 0, ""},
	Isc_gfix_pip_err:                                              {"\tNumber of inventory page errors\t: @1", 1, ""},
	Isc_gfix_rec_warn:                                             {"\tNumber of record level warnings\t: @1", 1, ""},
	Isc_gfix_blob_warn:                                            {"\tNumber of blob page warnings\t: @1", 1, ""},
	Isc_gfix_data_warn:                                            {"\tNumber of data page warnings\t: @1", 1, ""},
	Isc_gfix_index_warn:                                           {"\tNumber of index page warnings\t: @1", 1, ""},
	Isc_gfix_pointer_warn:                                         {"\tNumber of pointer page warnings\t: @1", 1, ""},
	Isc_gfix_trn_warn:                                             {"\tNumber of transaction page warnings\t: @1", 1, ""},
	Isc_gfix_db_warn:                                              {"\tNumber of database page warnings\t: @1", 1, ""},
	Isc_gfix_pip_warn:                                             {"\tNumber of inventory page warnings\t: @1", 1, ""},
	Isc_gfix_opt_icu:                                              {"   -icu                 fix database to be usable with present ICU version", 0, ""},
	Isc_gfix_opt_role:                                             {"   -role                set SQL role name", 0, ""},
	Isc_gfix_role_req:                                             {"SQL role name required", 0, ""},
	Isc_gfix_opt_repl:                                             {"   -repl(ica)           replica mode <none / read_only / read_write>", 0, ""},
	Isc_gfix_repl_mode_req:                                        {"replica mode (none / read_only / read_write) required", 0, ""},
	Isc_gfix_opt_parallel:                                         {"   -par(allel)          parallel workers <n> (-sweep, -icu)", 0, ""},
	Isc_gfix_opt_upgrade:                                          {"   -up(grade)           upgrade database ODS", 0, ""},
	Isc_gfix_overval_req:                                          {"too big numeric value", 0, "00000"},
	Isc_gfix_opt_sweep_timeout:                                    {"   -tim(eout)           set sweep interrupt timeout in seconds", 0, ""},
	Isc_gfix_opt_guid:                                             {"   -g(uid)              change a guid to the new one", 0, ""},
	335741930:                                                     {"number of threads required", 0, ""},
	Isc_gfix_opt_config_val:                                       {"   -conf(ig_validate)   validate configuration file <file_name>", 0, ""},
	335741932:                                                     {"Configuration file name required.", 0, ""},
	Isc_dsql_dbkey_from_non_table:                                 {"Cannot SELECT RDB$DB_KEY from a stored procedure.", 0, "HY000"},
	Isc_dsql_transitional_numeric:                                 {"Precision 10 to 18 changed from DOUBLE PRECISION in SQL dialect 1 to 64-bit scaled integer in SQL dialect 3", 0, "HY000"},
	Isc_dsql_dialect_warning_expr:                                 {"Use of @1 expression that returns different results in dialect 1 and dialect 3", 1, "01000"},
	Isc_sql_db_dialect_dtype_unsupport:                            {"Database SQL dialect @1 does not support reference to @2 datatype", 2, "HY000"},
	336003078:                                                     {"", 0, ""},
	Isc_sql_dialect_conflict_num:                                  {"DB dialect @1 and client dialect @2 conflict with respect to numeric precision @3.", 3, "HY000"},
	Isc_dsql_warning_number_ambiguous:                             {"WARNING: Numeric literal @1 is interpreted as a floating-point", 1, "HY104"},
	Isc_dsql_warning_number_ambiguous1:                            {"value in SQL dialect 1, but as an exact numeric value in SQL dialect 3.", 0, "HY104"},
	Isc_dsql_warn_precision_ambiguous:                             {"WARNING: NUMERIC and DECIMAL fields with precision 10 or greater are stored", 0, "HY104"},
	Isc_dsql_warn_precision_ambiguous1:                            {"as approximate floating-point values in SQL dialect 1, but as 64-bit", 0, "HY104"},
	Isc_dsql_warn_precision_ambiguous2:                            {"integers in SQL dialect 3.", 0, "HY104"},
	Isc_dsql_ambiguous_field_name:                                 {"Ambiguous field name between @1 and @2", 2, "42702"},
	Isc_dsql_udf_return_pos_err:                                   {"External function should have return position between 1 and @1", 1, "38000"},
	Isc_dsql_invalid_label:                                        {"Label @1 @2 in the current scope", 2, "HY000"},
	Isc_dsql_datatypes_not_comparable:                             {"Datatypes @1are not comparable in expression @2", 2, "HY004"},
	Isc_dsql_cursor_invalid:                                       {"Empty cursor name is not allowed", 0, "24000"},
	Isc_dsql_cursor_redefined:                                     {"Statement already has a cursor @1 assigned", 1, "24000"},
	Isc_dsql_cursor_not_found:                                     {"Cursor @1 is not found in the current context", 1, "34000"},
	Isc_dsql_cursor_exists:                                        {"Cursor @1 already exists in the current context", 1, "24000"},
	Isc_dsql_cursor_rel_ambiguous:                                 {"Relation @1 is ambiguous in cursor @2", 2, "34000"},
	Isc_dsql_cursor_rel_not_found:                                 {"Relation @1 is not found in cursor @2", 2, "34000"},
	Isc_dsql_cursor_not_open:                                      {"Cursor is not open", 0, "24000"},
	Isc_dsql_type_not_supp_ext_tab:                                {"Data type @1 is not supported for EXTERNAL TABLES. Relation @2, field @3", 3, "HY004"},
	Isc_dsql_feature_not_supported_ods:                            {"Feature not supported on ODS version older than @1.@2", 2, "0A000"},
	Isc_primary_key_required:                                      {"Primary key required on table @1", 1, "22000"},
	Isc_upd_ins_doesnt_match_pk:                                   {"UPDATE OR INSERT field list does not match primary key of table @1", 1, "42000"},
	Isc_upd_ins_doesnt_match_matching:                             {"UPDATE OR INSERT field list does not match MATCHING clause", 0, "42000"},
	Isc_upd_ins_with_complex_view:                                 {"UPDATE OR INSERT without MATCHING could not be used with views based on more than one table", 0, "54001"},
	Isc_dsql_incompatible_trigger_type:                            {"Incompatible trigger type", 0, "42000"},
	Isc_dsql_db_trigger_type_cant_change:                          {"Database trigger type can't be changed", 0, "42000"},
	Isc_dsql_record_version_table:                                 {"To be used with RDB$RECORD_VERSION, @1 must be a table or a view of single table", 1, "HY000"},
	Isc_dsql_invalid_sqlda_version:                                {"SQLDA version expected between @1 and @2, found @3", 3, "07002"},
	Isc_dsql_sqlvar_index:                                         {"at SQLVAR index @1", 1, "07002"},
	Isc_dsql_no_sqlind:                                            {"empty pointer to NULL indicator variable", 0, "07002"},
	Isc_dsql_no_sqldata:                                           {"empty pointer to data", 0, "07002"},
	Isc_dsql_no_input_sqlda:                                       {"No SQLDA for input values provided", 0, "07002"},
	Isc_dsql_no_output_sqlda:                                      {"No SQLDA for output values provided", 0, "07002"},
	Isc_dsql_wrong_param_num:                                      {"Wrong number of parameters (expected @1, got @2)", 2, "07001"},
	Isc_dsql_invalid_drop_ss_clause:                               {"Invalid DROP SQL SECURITY clause", 0, "42000"},
	Isc_upd_ins_cannot_default:                                    {"UPDATE OR INSERT value for field @1, part of the implicit or explicit MATCHING clause, cannot be DEFAULT", 1, "42000"},
	Isc_dsql_ltt_invalid_reference:                                {"LOCAL TEMPORARY TABLE @1 cannot be referenced in @2", 2, "42000"},
	Isc_dsql_using_statement_must_contain_clause:                  {"USING statement must contain at least one clause", 0, "42000"},
	336004072:                                                     {"Invalid DROP SQL SECURITY clause", 0, ""},
	336068609:                                                     {"ODS version not supported by DYN", 0, ""},
	336068610:                                                     {"unsupported DYN verb", 0, ""},
	336068611:                                                     {"STORE RDB$FIELD_DIMENSIONS failed", 0, ""},
	336068612:                                                     {"unsupported DYN verb", 0, ""},
	336068613:                                                     {"@1", 1, ""},
	336068614:                                                     {"unsupported DYN verb", 0, ""},
	336068615:                                                     {"DEFINE BLOB FILTER failed", 0, ""},
	336068616:                                                     {"DEFINE GENERATOR failed", 0, ""},
	336068617:                                                     {"DEFINE GENERATOR unexpected DYN verb", 0, ""},
	336068618:                                                     {"DEFINE FUNCTION failed", 0, ""},
	336068619:                                                     {"unsupported DYN verb", 0, ""},
	336068620:                                                     {"DEFINE FUNCTION ARGUMENT failed", 0, ""},
	336068621:                                                     {"STORE RDB$FIELDS failed", 0, ""},
	336068622:                                                     {"No table specified for index", 0, ""},
	336068623:                                                     {"STORE RDB$INDEX_SEGMENTS failed", 0, ""},
	336068624:                                                     {"unsupported DYN verb", 0, ""},
	336068625:                                                     {"PRIMARY KEY column lookup failed", 0, ""},
	336068626:                                                     {"could not find UNIQUE or PRIMARY KEY constraint in table @1 with specified columns", 1, ""},
	336068627:                                                     {"PRIMARY KEY lookup failed", 0, ""},
	336068628:                                                     {"could not find PRIMARY KEY index in specified table @1", 1, ""},
	336068629:                                                     {"STORE RDB$INDICES failed", 0, ""},
	336068630:                                                     {"STORE RDB$FIELDS failed", 0, ""},
	336068631:                                                     {"STORE RDB$RELATION_FIELDS failed", 0, ""},
	336068632:                                                     {"STORE RDB$RELATIONS failed", 0, ""},
	336068633:                                                     {"STORE RDB$USER_PRIVILEGES failed defining a table", 0, ""},
	336068634:                                                     {"unsupported DYN verb", 0, ""},
	336068635:                                                     {"STORE RDB$RELATIONS failed", 0, ""},
	336068636:                                                     {"STORE RDB$FIELDS failed", 0, ""},
	336068637:                                                     {"STORE RDB$RELATION_FIELDS failed", 0, ""},
	336068638:                                                     {"unsupported DYN verb", 0, ""},
	336068639:                                                     {"DEFINE TRIGGER failed", 0, ""},
	336068640:                                                     {"unsupported DYN verb", 0, ""},
	336068641:                                                     {"DEFINE TRIGGER MESSAGE failed", 0, ""},
	336068642:                                                     {"STORE RDB$VIEW_RELATIONS failed", 0, ""},
	336068643:                                                     {"ERASE RDB$FIELDS failed", 0, ""},
	336068644:                                                     {"ERASE BLOB FILTER failed", 0, ""},
	Isc_dyn_filter_not_found:                                      {"BLOB Filter @1 not found", 1, "42000"},
	336068646:                                                     {"unsupported DYN verb", 0, ""},
	336068647:                                                     {"ERASE RDB$FUNCTION_ARGUMENTS failed", 0, ""},
	336068648:                                                     {"ERASE RDB$FUNCTIONS failed", 0, ""},
	Isc_dyn_func_not_found:                                        {"Function @1 not found", 1, "42000"},
	336068650:                                                     {"unsupported DYN verb", 0, ""},
	336068651:                                                     {"Domain @1 is used in table @2 (local name @3) and cannot be dropped", 3, ""},
	336068652:                                                     {"ERASE RDB$FIELDS failed", 0, ""},
	336068653:                                                     {"ERASE RDB$FIELDS failed", 0, ""},
	336068654:                                                     {"Column not found", 0, ""},
	336068655:                                                     {"ERASE RDB$INDICES failed", 0, ""},
	Isc_dyn_index_not_found:                                       {"Index not found", 0, "42000"},
	336068657:                                                     {"ERASE RDB$INDEX_SEGMENTS failed", 0, ""},
	336068658:                                                     {"No segments found for index", 0, ""},
	336068659:                                                     {"No table specified in ERASE RFR", 0, ""},
	336068660:                                                     {"Column @1 from table @2 is referenced in view @3", 3, ""},
	336068661:                                                     {"ERASE RDB$RELATION_FIELDS failed", 0, ""},
	Isc_dyn_view_not_found:                                        {"View @1 not found", 1, "42000"},
	336068663:                                                     {"Column not found for table", 0, ""},
	336068664:                                                     {"ERASE RDB$INDEX_SEGMENTS failed", 0, ""},
	336068665:                                                     {"ERASE RDB$INDICES failed", 0, ""},
	336068666:                                                     {"ERASE RDB$RELATION_FIELDS failed", 0, ""},
	336068667:                                                     {"ERASE RDB$VIEW_RELATIONS failed", 0, ""},
	336068668:                                                     {"ERASE RDB$RELATIONS failed", 0, ""},
	336068669:                                                     {"Table not found", 0, ""},
	336068670:                                                     {"ERASE RDB$USER_PRIVILEGES failed", 0, ""},
	336068671:                                                     {"ERASE RDB$FILES failed", 0, ""},
	336068672:                                                     {"unsupported DYN verb", 0, ""},
	336068673:                                                     {"ERASE RDB$TRIGGER_MESSAGES failed", 0, ""},
	336068674:                                                     {"ERASE RDB$TRIGGERS failed", 0, ""},
	336068675:                                                     {"Trigger not found", 0, ""},
	336068676:                                                     {"MODIFY RDB$VIEW_RELATIONS failed", 0, ""},
	336068677:                                                     {"unsupported DYN verb", 0, ""},
	336068678:                                                     {"TRIGGER NAME expected", 0, ""},
	336068679:                                                     {"ERASE TRIGGER MESSAGE failed", 0, ""},
	336068680:                                                     {"Trigger Message not found", 0, ""},
	336068681:                                                     {"unsupported DYN verb", 0, ""},
	336068682:                                                     {"ERASE RDB$SECURITY_CLASSES failed", 0, ""},
	336068683:                                                     {"Security class not found", 0, ""},
	336068684:                                                     {"unsupported DYN verb", 0, ""},
	336068685:                                                     {"SELECT RDB$USER_PRIVILEGES failed in grant", 0, ""},
	336068686:                                                     {"SELECT RDB$USER_PRIVILEGES failed in grant", 0, ""},
	336068687:                                                     {"STORE RDB$USER_PRIVILEGES failed in grant", 0, ""},
	336068688:                                                     {"Specified domain or source column does not exist", 0, ""},
	336068689:                                                     {"Generation of column name failed", 0, ""},
	336068690:                                                     {"Generation of index name failed", 0, ""},
	336068691:                                                     {"Generation of trigger name failed", 0, ""},
	336068692:                                                     {"MODIFY DATABASE failed", 0, ""},
	336068693:                                                     {"MODIFY RDB$CHARACTER_SETS failed", 0, ""},
	336068694:                                                     {"MODIFY RDB$COLLATIONS failed", 0, ""},
	336068695:                                                     {"MODIFY RDB$FIELDS failed", 0, ""},
	336068696:                                                     {"MODIFY RDB$BLOB_FILTERS failed", 0, ""},
	Isc_dyn_domain_not_found:                                      {"Domain not found", 0, "42000"},
	336068698:                                                     {"unsupported DYN verb", 0, ""},
	336068699:                                                     {"MODIFY RDB$INDICES failed", 0, ""},
	336068700:                                                     {"MODIFY RDB$FUNCTIONS failed", 0, ""},
	336068701:                                                     {"Index column not found", 0, ""},
	336068702:                                                     {"MODIFY RDB$GENERATORS failed", 0, ""},
	336068703:                                                     {"MODIFY RDB$RELATION_FIELDS failed", 0, ""},
	336068704:                                                     {"Local column @1 not found", 1, ""},
	336068705:                                                     {"add EXTERNAL FILE not allowed", 0, ""},
	336068706:                                                     {"drop EXTERNAL FILE not allowed", 0, ""},
	336068707:                                                     {"MODIFY RDB$RELATIONS failed", 0, ""},
	336068708:                                                     {"MODIFY RDB$PROCEDURE_PARAMETERS failed", 0, ""},
	336068709:                                                     {"Table column not found", 0, ""},
	336068710:                                                     {"MODIFY TRIGGER failed", 0, ""},
	336068711:                                                     {"TRIGGER NAME expected", 0, ""},
	336068712:                                                     {"unsupported DYN verb", 0, ""},
	336068713:                                                     {"MODIFY TRIGGER MESSAGE failed", 0, ""},
	336068714:                                                     {"Create metadata BLOB failed", 0, ""},
	336068715:                                                     {"Write metadata BLOB failed", 0, ""},
	336068716:                                                     {"Close metadata BLOB failed", 0, ""},
	Isc_dyn_cant_modify_auto_trig:                                 {"Triggers created automatically cannot be modified", 0, "42000"},
	336068718:                                                     {"unsupported DYN verb", 0, ""},
	336068719:                                                     {"ERASE RDB$USER_PRIVILEGES failed in revoke(1)", 0, ""},
	336068720:                                                     {"Access to RDB$USER_PRIVILEGES failed in revoke(2)", 0, ""},
	336068721:                                                     {"ERASE RDB$USER_PRIVILEGES failed in revoke (3)", 0, ""},
	336068722:                                                     {"Access to RDB$USER_PRIVILEGES failed in revoke (4)", 0, ""},
	336068723:                                                     {"CREATE VIEW failed", 0, ""},
	336068724:                                                     {" attempt to index BLOB column in INDEX @1", 1, ""},
	336068725:                                                     {" attempt to index array column in index @1", 1, ""},
	336068726:                                                     {"key size too big for index @1", 1, ""},
	336068727:                                                     {"no keys for index @1", 1, ""},
	336068728:                                                     {"Unknown columns in index @1", 1, ""},
	336068729:                                                     {"STORE RDB$RELATION_CONSTRAINTS failed", 0, ""},
	336068730:                                                     {"STORE RDB$CHECK_CONSTRAINTS failed", 0, ""},
	336068731:                                                     {"Column: @1 not defined as NOT NULL - cannot be used in PRIMARY KEY constraint definition", 1, ""},
	336068732:                                                     {"A column name is repeated in the definition of constraint: @1", 1, ""},
	336068733:                                                     {"Integrity Constraint lookup failed", 0, ""},
	336068734:                                                     {"Same set of columns cannot be used in more than one PRIMARY KEY and/or UNIQUE constraint definition", 0, ""},
	336068735:                                                     {"STORE RDB$REF_CONSTRAINTS failed", 0, ""},
	336068736:                                                     {"No table specified in delete_constraint", 0, ""},
	336068737:                                                     {"ERASE RDB$RELATION_CONSTRAINTS failed", 0, ""},
	336068738:                                                     {"CONSTRAINT @1 does not exist.", 1, ""},
	336068739:                                                     {"Generation of constraint name failed", 0, ""},
	Isc_dyn_dup_table:                                             {"Table @1 already exists", 1, "42S01"},
	336068741:                                                     {"Number of referencing columns do not equal number of referenced columns", 0, ""},
	336068742:                                                     {"STORE RDB$PROCEDURES failed", 0, ""},
	Isc_dyn_dup_procedure:                                         {"Procedure @1 already exists", 1, ""},
	336068744:                                                     {"STORE RDB$PROCEDURE_PARAMETERS failed", 0, ""},
	336068745:                                                     {"Store into system table @1 failed", 1, ""},
	336068746:                                                     {"ERASE RDB$PROCEDURE_PARAMETERS failed", 0, ""},
	336068747:                                                     {"ERASE RDB$PROCEDURES failed", 0, ""},
	Isc_dyn_proc_not_found:                                        {"Procedure @1 not found", 1, "42000"},
	336068749:                                                     {"MODIFY RDB$PROCEDURES failed", 0, ""},
	336068750:                                                     {"DEFINE EXCEPTION failed", 0, ""},
	336068751:                                                     {"ERASE EXCEPTION failed", 0, ""},
	Isc_dyn_exception_not_found:                                   {"Exception not found", 0, "42000"},
	336068753:                                                     {"MODIFY EXCEPTION failed", 0, ""},
	Isc_dyn_proc_param_not_found:                                  {"Parameter @1 in procedure @2 not found", 2, "42000"},
	Isc_dyn_trig_not_found:                                        {"Trigger @1 not found", 1, "42000"},
	336068756:                                                     {"Only one data type change to the domain @1 allowed at a time", 1, ""},
	336068757:                                                     {"Only one data type change to the field @1 allowed at a time", 1, ""},
	336068758:                                                     {"STORE RDB$FILES failed", 0, ""},
	Isc_dyn_charset_not_found:                                     {"Character set @1 not found", 1, "42000"},
	Isc_dyn_collation_not_found:                                   {"Collation @1 not found", 1, "42000"},
	336068761:                                                     {"ERASE RDB$LOG_FILES failed", 0, ""},
	336068762:                                                     {"STORE RDB$LOG_FILES failed", 0, ""},
	Isc_dyn_role_not_found:                                        {"Role @1 not found", 1, "42000"},
	336068764:                                                     {"Difference file lookup failed", 0, ""},
	336068765:                                                     {"DEFINE SHADOW failed", 0, ""},
	336068766:                                                     {"MODIFY RDB$ROLES failed", 0, ""},
	Isc_dyn_name_longer:                                           {"Name longer than database column size", 0, "42000"},
	336068768:                                                     {"\"Only one constraint allowed for a domain\"", 0, ""},
	336068770:                                                     {"Looking up column position failed", 0, ""},
	336068771:                                                     {"A node name is not permitted in a table with external file definition", 0, ""},
	336068772:                                                     {"Shadow lookup failed", 0, ""},
	336068773:                                                     {"Shadow @1 already exists", 1, ""},
	336068774:                                                     {"Cannot add file with the same name as the database or added files", 0, ""},
	336068775:                                                     {"no grant option for privilege @1 on column @2 of table/view @3", 3, ""},
	336068776:                                                     {"no grant option for privilege @1 on column @2 of base table/view @3", 3, ""},
	336068777:                                                     {"no grant option for privilege @1 on table/view @2 (for column @3)", 3, ""},
	336068778:                                                     {"no grant option for privilege @1 on base table/view @2 (for column @3)", 3, ""},
	336068779:                                                     {"no @1 privilege with grant option on table/view @2 (for column @3)", 3, ""},
	336068780:                                                     {"no @1 privilege with grant option on base table/view @2 (for column @3)", 3, ""},
	336068781:                                                     {"no grant option for privilege @1 on table/view @2", 2, ""},
	336068782:                                                     {"no @1 privilege with grant option on table/view @2", 2, ""},
	336068783:                                                     {"table/view @1 does not exist", 1, ""},
	Isc_dyn_column_does_not_exist:                                 {"column @1 does not exist in table/view @2", 2, "42S22"},
	336068785:                                                     {"Can not alter a view", 0, ""},
	336068786:                                                     {"EXTERNAL FILE table not supported in this context", 0, ""},
	336068787:                                                     {"attempt to index COMPUTED BY column in INDEX @1", 1, ""},
	336068788:                                                     {"Table Name lookup failed", 0, ""},
	336068789:                                                     {"attempt to index a view", 0, ""},
	336068790:                                                     {"SELECT RDB$RELATIONS failed in grant", 0, ""},
	336068791:                                                     {"SELECT RDB$RELATION_FIELDS failed in grant", 0, ""},
	336068792:                                                     {"SELECT RDB$RELATIONS/RDB$OWNER_NAME failed in grant", 0, ""},
	336068793:                                                     {"SELECT RDB$USER_PRIVILEGES failed in grant", 0, ""},
	336068794:                                                     {"SELECT RDB$VIEW_RELATIONS/RDB$RELATION_FIELDS/... failed in grant", 0, ""},
	336068795:                                                     {"column @1 from table @2 is referenced in index @3", 3, ""},
	Isc_dyn_role_does_not_exist:                                   {"SQL role @1 does not exist", 1, "28000"},
	Isc_dyn_no_grant_admin_opt:                                    {"user @1 has no grant admin option on SQL role @2", 2, "28000"},
	Isc_dyn_user_not_role_member:                                  {"user @1 is not a member of SQL role @2", 2, "28000"},
	Isc_dyn_delete_role_failed:                                    {"@1 is not the owner of SQL role @2", 2, "28000"},
	Isc_dyn_grant_role_to_user:                                    {"@1 is a SQL role and not a user", 1, "28000"},
	Isc_dyn_inv_sql_role_name:                                     {"user name @1 could not be used for SQL role", 1, "28000"},
	Isc_dyn_dup_sql_role:                                          {"SQL role @1 already exists", 1, "42000"},
	Isc_dyn_kywd_spec_for_role:                                    {"keyword @1 can not be used as a SQL role name", 1, "28000"},
	Isc_dyn_roles_not_supported:                                   {"SQL roles are not supported in on older versions of the database.  A backup and restore of the database is required.", 0, "28000"},
	Isc_dyn_domain_name_exists:                                    {"Cannot rename domain @1 to @2.  A domain with that name already exists.", 2, "42000"},
	Isc_dyn_field_name_exists:                                     {"Cannot rename column @1 to @2.  A column with that name already exists in table @3.", 3, "42S21"},
	Isc_dyn_dependency_exists:                                     {"Column @1 from table @2 is referenced in @3", 3, "42000"},
	Isc_dyn_dtype_invalid:                                         {"Cannot change datatype for column @1.  Changing datatype is not supported for BLOB or ARRAY columns.", 1, "42000"},
	Isc_dyn_char_fld_too_small:                                    {"New size specified for column @1 must be at least @2 characters.", 2, "42000"},
	Isc_dyn_invalid_dtype_conversion:                              {"Cannot change datatype for @1.  Conversion from base type @2 to @3 is not supported.", 3, "42000"},
	Isc_dyn_dtype_conv_invalid:                                    {"Cannot change datatype for column @1 from a character type to a non-character type.", 1, "42000"},
	Isc_dyn_virmemexh:                                             {"unable to allocate memory from the operating system", 0, ""},
	Isc_dyn_zero_len_id:                                           {"Zero length identifiers are not allowed", 0, "42000"},
	Isc_del_gen_fail:                                              {"ERASE RDB$GENERATORS failed", 0, ""},
	Isc_dyn_gen_not_found:                                         {"Sequence @1 not found", 1, "42000"},
	336068823:                                                     {"Difference file is not defined", 0, ""},
	336068824:                                                     {"Difference file is already defined", 0, ""},
	336068825:                                                     {"Database is already in the physical backup mode", 0, ""},
	336068826:                                                     {"Database is not in the physical backup mode", 0, ""},
	336068827:                                                     {"DEFINE COLLATION failed", 0, ""},
	336068828:                                                     {"CREATE COLLATION statement is not supported in older versions of the database.  A backup and restore is required.", 0, ""},
	Isc_max_coll_per_charset:                                      {"Maximum number of collations per character set exceeded", 0, "2C000"},
	Isc_invalid_coll_attr:                                         {"Invalid collation attributes", 0, "HY000"},
	336068831:                                                     {"Collation @1 not installed for character set @2", 2, ""},
	336068832:                                                     {"Cannot use the internal domain @1 as new type for field @2", 2, ""},
	336068833:                                                     {"Default value is not allowed for array type in field @1", 1, ""},
	336068834:                                                     {"Default value is not allowed for array type in domain @1", 1, ""},
	336068835:                                                     {"DYN_UTIL_is_array failed for domain @1", 1, ""},
	336068836:                                                     {"DYN_UTIL_copy_domain failed for domain @1", 1, ""},
	336068837:                                                     {"Local column @1 doesn't have a default", 1, ""},
	336068838:                                                     {"Local column @1 default belongs to domain @2", 2, ""},
	336068839:                                                     {"File name is invalid", 0, ""},
	Isc_dyn_wrong_gtt_scope:                                       {"@1 cannot reference @2", 2, "HY000"},
	336068841:                                                     {"Local column @1 is computed, cannot set a default value", 1, ""},
	Isc_del_coll_fail:                                             {"ERASE RDB$COLLATIONS failed", 0, ""},
	Isc_dyn_coll_used_table:                                       {"Collation @1 is used in table @2 (field name @3) and cannot be dropped", 3, "HY000"},
	Isc_dyn_coll_used_domain:                                      {"Collation @1 is used in domain @2 and cannot be dropped", 2, "HY000"},
	Isc_dyn_cannot_del_syscoll:                                    {"Cannot delete system collation", 0, "HY000"},
	Isc_dyn_cannot_del_def_coll:                                   {"Cannot delete default collation of CHARACTER SET @1", 1, "HY000"},
	336068847:                                                     {"Domain @1 is used in procedure @2 (parameter name @3) and cannot be dropped", 3, ""},
	336068848:                                                     {"Field @1 cannot be used twice in index @2", 2, ""},
	Isc_dyn_table_not_found:                                       {"Table @1 not found", 1, "42000"},
	336068850:                                                     {"attempt to reference a view (@1) in a foreign key", 1, ""},
	Isc_dyn_coll_used_procedure:                                   {"Collation @1 is used in procedure @2 (parameter name @3) and cannot be dropped", 3, "HY000"},
	Isc_dyn_scale_too_big:                                         {"New scale specified for column @1 must be at most @2.", 2, "42000"},
	Isc_dyn_precision_too_small:                                   {"New precision specified for column @1 must be at least @2.", 2, "42000"},
	336068854:                                                     {"@1 is not grantor of @2 on @3 to @4.", 4, ""},
	Isc_dyn_miss_priv_warning:                                     {"Warning: @1 on @2 is not granted to @3.", 3, "42000"},
	Isc_dyn_ods_not_supp_feature:                                  {"Feature '@1' is not supported in ODS @2.@3", 3, "0A000"},
	Isc_dyn_cannot_addrem_computed:                                {"Cannot add or remove COMPUTED from column @1", 1, "42000"},
	Isc_dyn_no_empty_pw:                                           {"Password should not be empty string", 0, "42000"},
	Isc_dyn_dup_index:                                             {"Index @1 already exists", 1, "42S11"},
	Isc_dyn_locksmith_use_granted:                                 {"Only @1 or user with privilege USE_GRANTED_BY_CLAUSE can use GRANTED BY clause", 1, ""},
	Isc_dyn_dup_exception:                                         {"Exception @1 already exists", 1, ""},
	Isc_dyn_dup_generator:                                         {"Sequence @1 already exists", 1, ""},
	336068863:                                                     {"ERASE RDB$USER_PRIVILEGES failed in REVOKE ALL ON ALL", 0, ""},
	Isc_dyn_package_not_found:                                     {"Package @1 not found", 1, "42000"},
	Isc_dyn_schema_not_found:                                      {"Schema @1 not found", 1, "42000"},
	Isc_dyn_cannot_mod_sysproc:                                    {"Cannot ALTER or DROP system procedure @1", 1, "HY000"},
	Isc_dyn_cannot_mod_systrig:                                    {"Cannot ALTER or DROP system trigger @1", 1, "HY000"},
	Isc_dyn_cannot_mod_sysfunc:                                    {"Cannot ALTER or DROP system function @1", 1, "HY000"},
	Isc_dyn_invalid_ddl_proc:                                      {"Invalid DDL statement for procedure @1", 1, "HY000"},
	Isc_dyn_invalid_ddl_trig:                                      {"Invalid DDL statement for trigger @1", 1, "HY000"},
	Isc_dyn_funcnotdef_package:                                    {"Function @1 has not been defined on the package body @2", 2, "42000"},
	Isc_dyn_procnotdef_package:                                    {"Procedure @1 has not been defined on the package body @2", 2, "42000"},
	Isc_dyn_funcsignat_package:                                    {"Function @1 has a signature mismatch on package body @2", 2, "42000"},
	Isc_dyn_procsignat_package:                                    {"Procedure @1 has a signature mismatch on package body @2", 2, "42000"},
	Isc_dyn_defvaldecl_package_proc:                               {"Default values for parameters are not allowed in the definition of a previously declared packaged procedure @1.@2", 2, "42000"},
	Isc_dyn_dup_function:                                          {"Function @1 already exists", 1, ""},
	Isc_dyn_package_body_exists:                                   {"Package body @1 already exists", 1, "42000"},
	Isc_dyn_invalid_ddl_func:                                      {"Invalid DDL statement for function @1", 1, "HY000"},
	Isc_dyn_newfc_oldsyntax:                                       {"Cannot alter new style function @1 with ALTER EXTERNAL FUNCTION. Use ALTER FUNCTION instead.", 1, "42000"},
	336068880:                                                     {"Cannot delete system generator @1", 1, ""},
	336068881:                                                     {"Identity column @1 of table @2 must be of exact number type with zero scale", 2, ""},
	336068882:                                                     {"Identity column @1 of table @2 cannot be changed to NULLable", 2, ""},
	336068883:                                                     {"Identity column @1 of table @2 cannot have default value", 2, ""},
	336068884:                                                     {"Domain @1 must be of exact number type with zero scale because it's used in an identity column", 1, ""},
	336068885:                                                     {"Generation of generator name failed", 0, ""},
	Isc_dyn_func_param_not_found:                                  {"Parameter @1 in function @2 not found", 2, "42000"},
	Isc_dyn_routine_param_not_found:                               {"Parameter @1 of routine @2 not found", 2, "42000"},
	Isc_dyn_routine_param_ambiguous:                               {"Parameter @1 of routine @2 is ambiguous (found in both procedures and functions). Use a specifier keyword.", 2, "42000"},
	Isc_dyn_coll_used_function:                                    {"Collation @1 is used in function @2 (parameter name @3) and cannot be dropped", 3, "HY000"},
	Isc_dyn_domain_used_function:                                  {"Domain @1 is used in function @2 (parameter name @3) and cannot be dropped", 3, "HY000"},
	Isc_dyn_alter_user_no_clause:                                  {"ALTER USER requires at least one clause to be specified", 0, "42000"},
	336068892:                                                     {"Cannot delete system SQL role @1", 1, ""},
	336068893:                                                     {"Column @1 is not an identity column", 1, ""},
	Isc_dyn_duplicate_package_item:                                {"Duplicate @1 @2", 2, "42000"},
	Isc_dyn_cant_modify_sysobj:                                    {"System @1 @2 cannot be modified", 2, "42000"},
	Isc_dyn_cant_use_zero_increment:                               {"INCREMENT BY 0 is an illegal option for sequence @1", 1, "42000"},
	Isc_dyn_cant_use_in_foreignkey:                                {"Can't use @1 in FOREIGN KEY constraint", 1, "42000"},
	Isc_dyn_defvaldecl_package_func:                               {"Default values for parameters are not allowed in the definition of a previously declared packaged function @1.@2", 2, "42000"},
	Isc_dyn_create_user_no_password:                               {"Password must be specified when creating user", 0, ""},
	Isc_dyn_cyclic_role:                                           {"role @1 can not be granted to role @2", 2, "42000"},
	336068901:                                                     {"DROP SYSTEM PRIVILEGES should not be used in CREATE ROLE operator", 0, ""},
	336068902:                                                     {"Access to SYSTEM PRIVILEGES in ROLES denied to @1", 1, ""},
	336068903:                                                     {"Only @1, DB owner @2 or user with privilege USE_GRANTED_BY_CLAUSE can use GRANTED BY clause", 2, ""},
	Isc_dyn_cant_use_zero_inc_ident:                               {"INCREMENT BY 0 is an illegal option for identity column @1 of table @2", 2, "42000"},
	Isc_dyn_concur_alter_database:                                 {"Concurrent ALTER DATABASE is not supported", 0, ""},
	Isc_dyn_incompat_alter_database:                               {"Incompatible ALTER DATABASE clauses: '@1' and '@2'", 2, ""},
	Isc_dyn_no_ddl_grant_opt_priv:                                 {"no @1 privilege with grant option on DDL @2", 2, "42000"},
	Isc_dyn_no_grant_opt_priv:                                     {"no @1 privilege with grant option on object @2", 2, "42000"},
	Isc_dyn_func_not_exist:                                        {"Function @1 does not exist", 1, "42000"},
	Isc_dyn_proc_not_exist:                                        {"Procedure @1 does not exist", 1, "42000"},
	Isc_dyn_pack_not_exist:                                        {"Package @1 does not exist", 1, "42000"},
	Isc_dyn_trig_not_exist:                                        {"Trigger @1 does not exist", 1, "42000"},
	Isc_dyn_view_not_exist:                                        {"View @1 does not exist", 1, "42000"},
	Isc_dyn_rel_not_exist:                                         {"Table @1 does not exist", 1, "42000"},
	Isc_dyn_exc_not_exist:                                         {"Exception @1 does not exist", 1, "42000"},
	Isc_dyn_gen_not_exist:                                         {"Generator/Sequence @1 does not exist", 1, "42000"},
	Isc_dyn_fld_not_exist:                                         {"Field @1 of table @2 does not exist", 2, "42000"},
	Isc_dyn_dup_trigger:                                           {"Trigger @1 already exists", 1, ""},
	Isc_dyn_dup_domain:                                            {"Domain @1 already exists", 1, ""},
	Isc_dyn_dup_collation:                                         {"Collation @1 already exists", 1, ""},
	Isc_dyn_dup_package:                                           {"Package @1 already exists", 1, ""},
	Isc_dyn_index_schema_must_match_table:                         {"Index schema (@1) must match table schema (@2)", 2, "42000"},
	Isc_dyn_trig_schema_must_match_table:                          {"Trigger schema (@1) must match table schema (@2)", 2, "42000"},
	Isc_dyn_dup_schema:                                            {"Schema @1 already exists", 1, ""},
	Isc_dyn_cannot_mod_system_schema:                              {"Cannot ALTER or DROP SYSTEM schema", 0, "28000"},
	Isc_dyn_cannot_drop_non_emptyschema:                           {"Cannot DROP schema @1 because it has objects", 1, "HY000"},
	Isc_dyn_cannot_mod_obj_sys_schema:                             {"Cannot CREATE/ALTER/DROP @1 in SYSTEM schema", 1, "28000"},
	Isc_dyn_cannot_create_reserved_schema:                         {"Schema name @1 is reserved and cannot be created", 1, "HY000"},
	Isc_dyn_cannot_infer_schema:                                   {"Cannot infer schema name as there is no valid schema in the search path", 0, "42000"},
	Isc_dyn_dup_blob_filter:                                       {"Blob filter @1 already exists", 1, ""},
	Isc_dyn_column_name_exists:                                    {"Column @1 already exists in table @2", 2, "42000"},
	Isc_dyn_constant_not_found:                                    {"Constant @1 not found", 1, ""},
	Isc_dyn_dup_const:                                             {"Constant @1 already exists", 1, ""},
	Isc_dyn_non_constant_constant:                                 {"The constant @1 must be initialized by a constant expression", 1, ""},
	Isc_dyn_function_mismatch:                                     {"Function @1 cannot change between aggregate and non-aggregate", 1, "42000"},
	336069608:                                                     {"role @1 can not be granted to role @2", 2, ""},
	Isc_dyn_dup_domain_rdb:                                        {"Domain @1 already exists", 1, "42000"},
	Isc_dyn_dup_trigger_rdb:                                       {"Trigger @1 already exists", 1, "42000"},
	Isc_dyn_dup_package_rdb:                                       {"Package @1 already exists", 1, "42000"},
	Isc_dyn_dup_filter:                                            {"Filter @1 already exists", 1, "42000"},
	Isc_dyn_job_not_found:                                         {"Job @1 not found", 1, "42000"},
	Isc_dyn_invalid_ddl_job:                                       {"Invalid DDL statement for job @1", 1, "HY000"},
	Isc_dyn_alter_job_no_clause:                                   {"ALTER JOB requires at least one clause to be specified", 0, "42000"},
	Isc_dyn_cant_modify_collation:                                 {"Cannot change collation of field '@1.@2' to @3", 3, "42000"},
	Isc_dyn_foreign_server_not_found:                              {"Foreign server @1 not found", 1, "42000"},
	Isc_dyn_user_mapping_not_found:                                {"User mapping for user @1 and server @2 not found", 2, "42000"},
	Isc_dyn_job_already_exists:                                    {"Job @1 already exists", 1, "42000"},
	Isc_dyn_foreign_server_already_exists:                         {"Foreign server @1 already exists", 1, "42000"},
	336069626:                                                     {"attempt to index a foreign table", 0, ""},
	336330752:                                                     {"could not locate appropriate error message", 0, ""},
	Isc_gbak_unknown_switch:                                       {"found unknown switch", 0, "00000"},
	Isc_gbak_page_size_missing:                                    {"page size parameter missing", 0, "00000"},
	Isc_gbak_page_size_toobig:                                     {"Page size specified (@1) greater than limit (32768 bytes)", 1, "00000"},
	Isc_gbak_redir_ouput_missing:                                  {"redirect location for output is not specified", 0, "00000"},
	Isc_gbak_switches_conflict:                                    {"conflicting switches for backup/restore", 0, "00000"},
	Isc_gbak_unknown_device:                                       {"device type @1 not known", 1, "00000"},
	Isc_gbak_no_protection:                                        {"protection is not there yet", 0, "00000"},
	Isc_gbak_page_size_not_allowed:                                {"page size is allowed only on restore or create", 0, "00000"},
	Isc_gbak_multi_source_dest:                                    {"multiple sources or destinations specified", 0, "00000"},
	Isc_gbak_filename_missing:                                     {"requires both input and output filenames", 0, "00000"},
	Isc_gbak_dup_inout_names:                                      {"input and output have the same name.  Disallowed.", 0, "00000"},
	Isc_gbak_inv_page_size:                                        {"expected page size, encountered \"@1\"", 1, "00000"},
	Isc_gbak_db_specified:                                         {"REPLACE specified, but the first file @1 is a database", 1, "00000"},
	Isc_gbak_db_exists:                                            {"database @1 already exists.  To replace it, use the -REP switch", 1, "00000"},
	Isc_gbak_unk_device:                                           {"device type not specified", 0, "00000"},
	336330768:                                                     {"cannot create APOLLO tape descriptor file @1", 1, ""},
	336330769:                                                     {"cannot set APOLLO tape descriptor attribute for @1", 1, ""},
	336330770:                                                     {"cannot create APOLLO cartridge descriptor file @1", 1, ""},
	336330771:                                                     {"cannot close APOLLO tape descriptor file @1", 1, ""},
	Isc_gbak_blob_info_failed:                                     {"gds_$blob_info failed", 0, "00000"},
	Isc_gbak_unk_blob_item:                                        {"do not understand BLOB INFO item @1", 1, "00000"},
	Isc_gbak_get_seg_failed:                                       {"gds_$get_segment failed", 0, "00000"},
	Isc_gbak_close_blob_failed:                                    {"gds_$close_blob failed", 0, "00000"},
	Isc_gbak_open_blob_failed:                                     {"gds_$open_blob failed", 0, "00000"},
	Isc_gbak_put_blr_gen_id_failed:                                {"Failed in put_blr_gen_id", 0, "00000"},
	Isc_gbak_unk_type:                                             {"data type @1 not understood", 1, "00000"},
	Isc_gbak_comp_req_failed:                                      {"gds_$compile_request failed", 0, "00000"},
	Isc_gbak_start_req_failed:                                     {"gds_$start_request failed", 0, "00000"},
	Isc_gbak_rec_failed:                                           {"gds_$receive failed", 0, "00000"},
	Isc_gbak_rel_req_failed:                                       {"gds_$release_request failed", 0, "00000"},
	Isc_gbak_db_info_failed:                                       {"gds_$database_info failed", 0, "00000"},
	Isc_gbak_no_db_desc:                                           {"Expected database description record", 0, "00000"},
	Isc_gbak_db_create_failed:                                     {"failed to create database @1", 1, "00000"},
	Isc_gbak_decomp_len_error:                                     {"RESTORE: decompression length error", 0, "00000"},
	Isc_gbak_tbl_missing:                                          {"cannot find table @1", 1, "00000"},
	Isc_gbak_blob_col_missing:                                     {"Cannot find column for BLOB", 0, "00000"},
	Isc_gbak_create_blob_failed:                                   {"gds_$create_blob failed", 0, "00000"},
	Isc_gbak_put_seg_failed:                                       {"gds_$put_segment failed", 0, "00000"},
	Isc_gbak_rec_len_exp:                                          {"expected record length", 0, "00000"},
	Isc_gbak_inv_rec_len:                                          {"wrong length record, expected @1 encountered @2", 2, "00000"},
	Isc_gbak_exp_data_type:                                        {"expected data attribute", 0, "00000"},
	Isc_gbak_gen_id_failed:                                        {"Failed in store_blr_gen_id", 0, "00000"},
	Isc_gbak_unk_rec_type:                                         {"do not recognize record type @1", 1, "00000"},
	Isc_gbak_inv_bkup_ver:                                         {"Expected backup version 1..10.  Found @1", 1, "00000"},
	Isc_gbak_missing_bkup_desc:                                    {"expected backup description record", 0, "00000"},
	Isc_gbak_string_trunc:                                         {"string truncated", 0, "00000"},
	Isc_gbak_cant_rest_record:                                     {"warning -- record could not be restored", 0, "00000"},
	Isc_gbak_send_failed:                                          {"gds_$send failed", 0, "00000"},
	Isc_gbak_no_tbl_name:                                          {"no table name for data", 0, "00000"},
	Isc_gbak_unexp_eof:                                            {"unexpected end of file on backup file", 0, "00000"},
	Isc_gbak_db_format_too_old:                                    {"database format @1 is too old to restore to", 1, "00000"},
	Isc_gbak_inv_array_dim:                                        {"array dimension for column @1 is invalid", 1, "00000"},
	336330805:                                                     {"expected array version number @1 but instead found @2", 2, ""},
	336330806:                                                     {"expected array dimension @1 but instead found @2", 2, ""},
	Isc_gbak_xdr_len_expected:                                     {"Expected XDR record length", 0, "00000"},
	336330808:                                                     {"Unexpected I/O error while @1 backup file", 1, ""},
	336330809:                                                     {"adding file @1, starting at page @2", 2, ""},
	336330810:                                                     {"array", 0, ""},
	336330811:                                                     {"backup", 0, ""},
	336330812:                                                     {"    @1B(ACKUP_DATABASE) [O(VERWRITE)] backup database to file (replace existing\n\t\t\t\tfile if OVERWRITE used)", 1, ""},
	336330813:                                                     {"\t\tbackup file is compressed", 0, ""},
	336330814:                                                     {"    @1D(EVICE)             backup file device type on APOLLO (CT or MT)", 1, ""},
	336330815:                                                     {"    @1M(ETA_DATA)          backup or restore metadata only", 1, ""},
	336330816:                                                     {"blob", 0, ""},
	Isc_gbak_open_bkup_error:                                      {"cannot open backup file @1", 1, "00000"},
	Isc_gbak_open_error:                                           {"cannot open status and error output file @1", 1, "00000"},
	336330819:                                                     {"closing file, committing, and finishing", 0, ""},
	336330820:                                                     {"committing metadata", 0, ""},
	336330821:                                                     {"commit failed on table @1", 1, ""},
	336330822:                                                     {"committing secondary files", 0, ""},
	336330823:                                                     {"creating index @1", 1, ""},
	336330824:                                                     {"committing data for table @1", 1, ""},
	336330825:                                                     {"    @1C(REATE_DATABASE)    create database from backup file (restore)", 1, ""},
	336330826:                                                     {"created database @1, page_size @2 bytes", 2, ""},
	336330827:                                                     {"creating file @1", 1, ""},
	336330828:                                                     {"creating indexes", 0, ""},
	336330829:                                                     {"database @1 has a page size of @2 bytes.", 2, ""},
	336330830:                                                     {"    @1I(NACTIVE)           deactivate indexes during restore", 1, ""},
	336330831:                                                     {"do not understand BLOB INFO item @1", 1, ""},
	336330832:                                                     {"do not recognize @1 attribute @2 -- continuing", 2, ""},
	336330833:                                                     {"error accessing BLOB column @1 -- continuing", 1, ""},
	336330834:                                                     {"Exiting before completion due to errors", 0, ""},
	336330835:                                                     {"Exiting before completion due to errors", 0, ""},
	336330836:                                                     {"column", 0, ""},
	336330837:                                                     {"file", 0, ""},
	336330838:                                                     {"file length", 0, ""},
	336330839:                                                     {"filter", 0, ""},
	336330840:                                                     {"finishing, closing, and going home", 0, ""},
	336330841:                                                     {"function", 0, ""},
	336330842:                                                     {"function argument", 0, ""},
	336330843:                                                     {"gbak version @1", 1, ""},
	336330844:                                                     {"domain", 0, ""},
	336330845:                                                     {"index", 0, ""},
	336330846:                                                     {"trigger @1 is invalid", 1, ""},
	336330847:                                                     {"legal switches are:", 0, ""},
	336330848:                                                     {"length given for initial file (@1) is less than minimum (@2)", 2, ""},
	336330849:                                                     {"    @1E(XPAND)             no data compression", 1, ""},
	336330850:                                                     {"    @1L(IMBO)              ignore transactions in limbo", 1, ""},
	336330851:                                                     {"    @1O(NE_AT_A_TIME)      restore one table at a time", 1, ""},
	336330852:                                                     {"opened file @1", 1, ""},
	336330853:                                                     {"    @1P(AGE_SIZE)          override default page size", 1, ""},
	336330854:                                                     {"page size", 0, ""},
	336330855:                                                     {"page size specified (@1 bytes) rounded up to @2 bytes", 2, ""},
	336330856:                                                     {"    @1Z                    print version number", 1, ""},
	336330857:                                                     {"privilege", 0, ""},
	336330858:                                                     {"     @1 records ignored", 1, ""},
	336330859:                                                     {"   @1 records restored", 1, ""},
	336330860:                                                     {"@1 records written", 1, ""},
	336330861:                                                     {"    @1Y  <path>            redirect/suppress status message output", 1, ""},
	336330862:                                                     {"Reducing the database page size from @1 bytes to @2 bytes", 2, ""},
	336330863:                                                     {"table", 0, ""},
	336330864:                                                     {"    @1REP(LACE_DATABASE)   replace database from backup file (restore)", 1, ""},
	336330865:                                                     {"    @1V(ERIFY)             report each action taken", 1, ""},
	336330866:                                                     {"restore failed for record in table @1", 1, ""},
	336330867:                                                     {"    restoring column @1", 1, ""},
	336330868:                                                     {"    restoring file @1", 1, ""},
	336330869:                                                     {"    restoring filter @1", 1, ""},
	336330870:                                                     {"restoring function @1", 1, ""},
	336330871:                                                     {"    restoring argument for function @1", 1, ""},
	336330872:                                                     {"     restoring gen id value of: @1", 1, ""},
	336330873:                                                     {"restoring domain @1", 1, ""},
	336330874:                                                     {"    restoring index @1 for table @2", 2, ""},
	336330875:                                                     {"restoring @1 privilege(s) for user @2", 2, ""},
	336330876:                                                     {"restoring data for table @1", 1, ""},
	336330877:                                                     {"restoring security class @1", 1, ""},
	336330878:                                                     {"    restoring trigger @1", 1, ""},
	336330879:                                                     {"    restoring trigger message for @1", 1, ""},
	336330880:                                                     {"    restoring type @1 for column @2", 2, ""},
	336330881:                                                     {"started transaction", 0, ""},
	336330882:                                                     {"starting transaction", 0, ""},
	336330883:                                                     {"security class", 0, ""},
	336330884:                                                     {"switches can be abbreviated to the unparenthesized characters", 0, ""},
	336330885:                                                     {"transportable backup -- data in XDR format", 0, ""},
	336330886:                                                     {"trigger", 0, ""},
	336330887:                                                     {"trigger message", 0, ""},
	336330888:                                                     {"trigger type", 0, ""},
	336330889:                                                     {"unknown switch \"@1\"", 1, ""},
	336330890:                                                     {"validation error on column in table @1", 1, ""},
	336330891:                                                     {"    Version(s) for database \"@1\"", 1, ""},
	336330892:                                                     {"view", 0, ""},
	336330893:                                                     {"    writing argument for function @1", 1, ""},
	336330894:                                                     {"    writing data for table @1", 1, ""},
	336330895:                                                     {"     writing gen id of: @1", 1, ""},
	336330896:                                                     {"         writing column @1", 1, ""},
	336330897:                                                     {"    writing filter @1", 1, ""},
	336330898:                                                     {"writing filters", 0, ""},
	336330899:                                                     {"    writing function @1", 1, ""},
	336330900:                                                     {"writing functions", 0, ""},
	336330901:                                                     {"    writing domain @1", 1, ""},
	336330902:                                                     {"writing domains", 0, ""},
	336330903:                                                     {"    writing index @1", 1, ""},
	336330904:                                                     {"writing @1 privilege(s) for user @2", 2, ""},
	336330905:                                                     {"    writing table @1", 1, ""},
	336330906:                                                     {"writing tables", 0, ""},
	336330907:                                                     {"    writing security class @1", 1, ""},
	336330908:                                                     {"    writing trigger @1", 1, ""},
	336330909:                                                     {"    writing trigger message for @1", 1, ""},
	336330910:                                                     {"writing trigger messages", 0, ""},
	336330911:                                                     {"writing triggers", 0, ""},
	336330912:                                                     {"    writing type @1 for column @2", 2, ""},
	336330913:                                                     {"writing types", 0, ""},
	336330914:                                                     {"writing shadow files", 0, ""},
	336330915:                                                     {"    writing shadow file @1", 1, ""},
	336330916:                                                     {"writing id generators", 0, ""},
	336330917:                                                     {"    writing generator @1 value @2", 2, ""},
	336330918:                                                     {"readied database @1 for backup", 1, ""},
	336330919:                                                     {"restoring table @1", 1, ""},
	336330920:                                                     {"type", 0, ""},
	336330921:                                                     {"gbak:", 0, ""},
	336330922:                                                     {"committing metadata for table @1", 1, ""},
	336330923:                                                     {"error committing metadata for table @1", 1, ""},
	336330924:                                                     {"    @1K(ILL)               restore without creating shadows", 1, ""},
	336330925:                                                     {"cannot commit index @1", 1, ""},
	336330926:                                                     {"cannot commit files", 0, ""},
	336330927:                                                     {"    @1T(RANSPORTABLE)      transportable backup -- data in XDR format", 1, ""},
	336330928:                                                     {"closing file, committing, and finishing. @1 bytes written", 1, ""},
	336330929:                                                     {"    @1G(ARBAGE_COLLECT)    inhibit garbage collection", 1, ""},
	336330930:                                                     {"    @1IG(NORE)             ignore bad checksums", 1, ""},
	336330931:                                                     {"\tcolumn @1 used in index @2 seems to have vanished", 2, ""},
	336330932:                                                     {"index @1 omitted because @2 of the expected @3 keys were found", 3, ""},
	336330933:                                                     {"    @1FA(CTOR)             blocking factor", 1, ""},
	Isc_gbak_missing_block_fac:                                    {"blocking factor parameter missing", 0, "00000"},
	Isc_gbak_inv_block_fac:                                        {"expected blocking factor, encountered \"@1\"", 1, "00000"},
	Isc_gbak_block_fac_specified:                                  {"a blocking factor may not be used in conjunction with device CT", 0, "00000"},
	336330937:                                                     {"restoring generator @1 value: @2", 2, ""},
	336330938:                                                     {"    @1OL(D_DESCRIPTIONS)   save old style metadata descriptions", 1, ""},
	336330939:                                                     {"    @1N(O_VALIDITY)        do not restore database validity conditions", 1, ""},
	Isc_gbak_missing_username:                                     {"user name parameter missing", 0, "00000"},
	Isc_gbak_missing_password:                                     {"password parameter missing", 0, "00000"},
	336330942:                                                     {"    @1PAS(SWORD)           Red Database password", 1, ""},
	336330943:                                                     {"    @1USER                 Red Database user name", 1, ""},
	336330944:                                                     {"writing stored procedures", 0, ""},
	336330945:                                                     {"writing stored procedure @1", 1, ""},
	336330946:                                                     {"writing parameter @1 for stored procedure", 1, ""},
	336330947:                                                     {"restoring stored procedure @1", 1, ""},
	336330948:                                                     {"    restoring parameter @1 for stored procedure", 1, ""},
	336330949:                                                     {"writing exceptions", 0, ""},
	336330950:                                                     {"writing exception @1", 1, ""},
	336330951:                                                     {"restoring exception @1", 1, ""},
	Isc_gbak_missing_skipped_bytes:                                {" missing parameter for the number of bytes to be skipped", 0, "00000"},
	Isc_gbak_inv_skipped_bytes:                                    {"expected number of bytes to be skipped, encountered \"@1\"", 1, "00000"},
	336330954:                                                     {"adjusting an invalid decompression length from @1 to @2", 2, ""},
	336330955:                                                     {"skipped @1 bytes after reading a bad attribute @2", 2, ""},
	336330956:                                                     {"    @1S(KIP_BAD_DATA)      skip number of bytes after reading bad data", 1, ""},
	336330957:                                                     {"skipped @1 bytes looking for next valid attribute, encountered attribute @2", 2, ""},
	336330958:                                                     {"writing table constraints", 0, ""},
	336330959:                                                     {"writing constraint @1", 1, ""},
	336330960:                                                     {"table constraint", 0, ""},
	336330961:                                                     {"writing referential constraints", 0, ""},
	336330962:                                                     {"writing check constraints", 0, ""},
	Isc_msgVerbose_write_charsets:                                 {"writing character sets", 0, ""},
	Isc_msgVerbose_write_collations:                               {"writing collations", 0, ""},
	Isc_gbak_err_restore_charset:                                  {"character set", 0, "00000"},
	Isc_msgVerbose_restore_charset:                                {"writing character set @1", 1, ""},
	Isc_gbak_err_restore_collation:                                {"collation", 0, "00000"},
	Isc_msgVerbose_restore_collation:                              {"writing collation @1", 1, ""},
	Isc_gbak_read_error:                                           {"Unexpected I/O error while reading from backup file", 0, "00000"},
	Isc_gbak_write_error:                                          {"Unexpected I/O error while writing to backup file", 0, "00000"},
	336330974:                                                     {"\n\nCould not open file name \"@1\"", 1, ""},
	336330975:                                                     {"\n\nCould not write to file \"@1\"", 1, ""},
	336330976:                                                     {"\n\nCould not read from file \"@1\"", 1, ""},
	336330977:                                                     {"Done with volume #@1, \"@2\"", 2, ""},
	336330978:                                                     {"\tPress return to reopen that file, or type a new\n\tname followed by return to open a different file.", 0, ""},
	336330979:                                                     {"Type a file name to open and hit return", 0, ""},
	336330980:                                                     {"  Name: ", 0, ""},
	336330981:                                                     {"\n\nERROR: Backup incomplete", 0, ""},
	336330982:                                                     {"Expected backup start time @1, found @2", 2, ""},
	336330983:                                                     {"Expected backup database @1, found @2", 2, ""},
	336330984:                                                     {"Expected volume number @1, found volume @2", 2, ""},
	Isc_gbak_db_in_use:                                            {"could not drop database @1 (no privilege or database might be in use)", 1, "00000"},
	336330986:                                                     {"Skipped bad security class entry: @1", 1, ""},
	336330987:                                                     {"Unknown V3 SUB_TYPE: @1 in FIELD: @2.", 2, ""},
	336330988:                                                     {"Converted V3 sub_type: @1 to character_set_id: @2 and collate_id: @3.", 3, ""},
	336330989:                                                     {"Converted V3 scale: @1 to character_set_id: @2 and callate_id: @3.", 3, ""},
	Isc_gbak_sysmemex:                                             {"System memory exhausted", 0, "00000"},
	336330991:                                                     {"    @1NT                   Non-Transportable backup file format", 1, ""},
	336330992:                                                     {"Index \"@1\" failed to activate because:", 1, ""},
	336330993:                                                     {"  The unique index has duplicate values or NULLs.", 0, ""},
	336330994:                                                     {"  Delete or Update duplicate values or NULLs, and activate index with", 0, ""},
	336330995:                                                     {"  ALTER INDEX \"@1\" ACTIVE;", 1, ""},
	336330996:                                                     {"  Not enough disk space to create the sort file for an index.", 0, ""},
	336330997:                                                     {"  Set the TMP environment variable to a directory on a filesystem that does have enough space, and activate index with", 0, ""},
	336330998:                                                     {"Database is not online due to failure to activate one or more indices.", 0, ""},
	336330999:                                                     {"Run gfix -online to bring database online without active indices.", 0, ""},
	Isc_write_role_1:                                              {"writing SQL roles", 0, ""},
	Isc_write_role_2:                                              {"    writing SQL role: @1", 1, ""},
	Isc_gbak_restore_role_failed:                                  {"SQL role", 0, "00000"},
	Isc_restore_role:                                              {"    restoring SQL role: @1", 1, ""},
	Isc_gbak_role_op:                                              {"    @1RO(LE)               Red Database SQL role", 1, ""},
	Isc_gbak_role_op_missing:                                      {"SQL role parameter missing", 0, "00000"},
	Isc_gbak_convert_ext_tables:                                   {"    @1CO(NVERT)            backup external files as tables", 1, ""},
	Isc_gbak_warning:                                              {"gbak: WARNING:", 0, ""},
	Isc_gbak_error:                                                {"gbak: ERROR:", 0, ""},
	Isc_gbak_page_buffers:                                         {"    @1BU(FFERS)            override page buffers default", 1, ""},
	Isc_gbak_page_buffers_missing:                                 {"page buffers parameter missing", 0, "00000"},
	Isc_gbak_page_buffers_wrong_param:                             {"expected page buffers, encountered \"@1\"", 1, "00000"},
	Isc_gbak_page_buffers_restore:                                 {"page buffers is allowed only on restore or create", 0, "00000"},
	336331013:                                                     {"Starting with volume #@1, \"@2\"", 2, ""},
	Isc_gbak_inv_size:                                             {"size specification either missing or incorrect for file @1", 1, "00000"},
	Isc_gbak_file_outof_sequence:                                  {"file @1 out of sequence", 1, "00000"},
	Isc_gbak_join_file_missing:                                    {"can't join -- one of the files missing", 0, "00000"},
	Isc_gbak_stdin_not_supptd:                                     {" standard input is not supported when using join operation", 0, "00000"},
	Isc_gbak_stdout_not_supptd:                                    {"standard output is not supported when using split operation or in verbose mode", 0, "00000"},
	Isc_gbak_bkup_corrupt:                                         {"backup file @1 might be corrupt", 1, "00000"},
	Isc_gbak_unk_db_file_spec:                                     {"database file specification missing", 0, "00000"},
	Isc_gbak_hdr_write_failed:                                     {"can't write a header record to file @1", 1, "00000"},
	Isc_gbak_disk_space_ex:                                        {"free disk space exhausted", 0, "00000"},
	Isc_gbak_size_lt_min:                                          {"file size given (@1) is less than minimum allowed (@2)", 2, "00000"},
	336331024:                                                     {"Warning -- free disk space exhausted for file @1, the rest of the bytes (@2) will be written to file @3", 3, ""},
	Isc_gbak_svc_name_missing:                                     {"service name parameter missing", 0, "00000"},
	Isc_gbak_not_ownr:                                             {"Cannot restore over current database, must be SYSDBA or owner of the existing database.", 0, "00000"},
	336331027:                                                     {"", 0, ""},
	336331028:                                                     {"    @1USE_(ALL_SPACE)      do not reserve space for record versions", 1, ""},
	336331029:                                                     {"    @1SE(RVICE)            use services manager", 1, ""},
	Isc_gbak_opt_mode:                                             {"    @1MO(DE) <access>      \"read_only\" or \"read_write\" access", 1, ""},
	Isc_gbak_mode_req:                                             {"\"read_only\" or \"read_write\" required", 0, "00000"},
	336331032:                                                     {"setting database to read-only access", 0, ""},
	Isc_gbak_just_data:                                            {"just data ignore all constraints etc.", 0, "00000"},
	Isc_gbak_data_only:                                            {"restoring data only ignoring foreign key, unique, not null & other constraints", 0, "00000"},
	336331035:                                                     {"closing file, committing, and finishing. @1 bytes written", 1, ""},
	336331036:                                                     {"    @1R(ECREATE_DATABASE) [O(VERWRITE)] create (or replace if OVERWRITE used)\\n\t\t\t\tdatabase from backup file (restore)", 1, ""},
	Isc_gbak_activating_idx:                                       {"    activating and creating deferred index @1", 1, ""},
	336331038:                                                     {"check constraint", 0, ""},
	336331039:                                                     {"exception", 0, ""},
	336331040:                                                     {"array dimensions", 0, ""},
	336331041:                                                     {"generator", 0, ""},
	336331042:                                                     {"procedure", 0, ""},
	336331043:                                                     {"procedure parameter", 0, ""},
	336331044:                                                     {"referential constraint", 0, ""},
	336331045:                                                     {"type (in RDB$TYPES)", 0, ""},
	336331046:                                                     {"    @1NOD(BTRIGGERS)       do not run database triggers", 1, ""},
	336331047:                                                     {"    @1TRU(STED)            use trusted authentication", 1, ""},
	Isc_write_map_1:                                               {"writing names mapping", 0, ""},
	Isc_write_map_2:                                               {"    writing map for @1", 1, ""},
	Isc_get_map_1:                                                 {"    restoring map for @1", 1, ""},
	Isc_get_map_2:                                                 {"name mapping", 0, ""},
	Isc_get_map_3:                                                 {"cannot restore arbitrary mapping", 0, ""},
	Isc_get_map_4:                                                 {"restoring names mapping", 0, ""},
	336331054:                                                     {"    @1FIX_FSS_D(ATA)       fix malformed UNICODE_FSS data", 1, ""},
	336331055:                                                     {"    @1FIX_FSS_M(ETADATA)   fix malformed UNICODE_FSS metadata", 1, ""},
	336331056:                                                     {"Character set parameter missing", 0, ""},
	336331057:                                                     {"Character set @1 not found", 1, ""},
	336331058:                                                     {"    @1FE(TCH_PASSWORD)     fetch password from file", 1, ""},
	336331059:                                                     {"too many passwords provided", 0, ""},
	336331060:                                                     {"could not open password file @1, errno @2", 2, ""},
	336331061:                                                     {"could not read password file @1, errno @2", 2, ""},
	336331062:                                                     {"empty password file @1", 1, ""},
	336331063:                                                     {"Attribute @1 was already processed for exception @2", 2, ""},
	336331064:                                                     {"Skipping attribute @1 because the message already exists for exception @2", 2, ""},
	336331065:                                                     {"Trying to recover from unexpected attribute @1 due to wrong message length for exception @2", 2, ""},
	336331066:                                                     {"Attribute not specified for storing text bigger than 255 bytes", 0, ""},
	336331067:                                                     {"Unable to store text bigger than 65536 bytes", 0, ""},
	336331068:                                                     {"Failed while adjusting the security class name", 0, ""},
	336331069:                                                     {"Usage:", 0, ""},
	336331070:                                                     {"     gbak -b <db set> <backup set> [backup options] [general options]", 0, ""},
	336331071:                                                     {"     gbak -c <backup set> <db set> [restore options] [general options]", 0, ""},
	336331072:                                                     {"     <db set> = <database> | <db1 size1>...<dbN> (size in db pages)", 0, ""},
	336331073:                                                     {"     <backup set> = <backup> | <bk1 size1>...<bkN> (size in bytes = n[K|M|G])", 0, ""},
	336331074:                                                     {"     -recreate overwrite and -replace can be used instead of -c", 0, ""},
	336331075:                                                     {"backup options are:", 0, ""},
	336331076:                                                     {"restore options are:", 0, ""},
	336331077:                                                     {"general options are:", 0, ""},
	Isc_gbak_missing_interval:                                     {"verbose interval value parameter missing", 0, "00000"},
	Isc_gbak_wrong_interval:                                       {"verbose interval value cannot be smaller than @1", 1, "00000"},
	336331080:                                                     {"    @1VERBI(NT) <n>        verbose information with explicit interval", 1, ""},
	Isc_gbak_verify_verbint:                                       {"verify (verbose) and verbint options are mutually exclusive", 0, "00000"},
	Isc_gbak_option_only_restore:                                  {"option -@1 is allowed only on restore or create", 1, "00000"},
	Isc_gbak_option_only_backup:                                   {"option -@1 is allowed only on backup", 1, "00000"},
	Isc_gbak_option_conflict:                                      {"options -@1 and -@2 are mutually exclusive", 2, "00000"},
	Isc_gbak_param_conflict:                                       {"parameter for option -@1 was already specified with value \"@2\"", 2, "00000"},
	Isc_gbak_option_repeated:                                      {"option -@1 was already specified", 1, "00000"},
	336331087:                                                     {"writing package @1", 1, ""},
	336331088:                                                     {"writing packages", 0, ""},
	336331089:                                                     {"restoring package @1", 1, ""},
	336331090:                                                     {"package", 0, ""},
	Isc_gbak_max_dbkey_recursion:                                  {"dependency depth greater than @1 for view @2", 2, "00000"},
	Isc_gbak_max_dbkey_length:                                     {"value greater than @1 when calculating length of rdb$db_key for view @2", 2, "00000"},
	Isc_gbak_invalid_metadata:                                     {"Invalid metadata detected. Use -FIX_FSS_METADATA option.", 0, "00000"},
	Isc_gbak_invalid_data:                                         {"Invalid data detected. Use -FIX_FSS_DATA option.", 0, "00000"},
	336331095:                                                     {"text for attribute @1 is too large in @2, truncating to @3 bytes", 3, ""},
	Isc_gbak_inv_bkup_ver2:                                        {"Expected backup version @2..@3.  Found @1", 3, "00000"},
	336331097:                                                     {"    writing view @1", 1, ""},
	336331098:                                                     {"    table @1 is a view", 1, ""},
	336331099:                                                     {"writing security classes", 0, ""},
	Isc_gbak_db_format_too_old2:                                   {"database format @1 is too old to backup", 1, "00000"},
	336331101:                                                     {"backup version is @1", 1, ""},
	336331102:                                                     {"adjusting system generators", 0, ""},
	336331103:                                                     {"Error closing database, but backup file is OK", 0, ""},
	336331104:                                                     {"database", 0, ""},
	336331105:                                                     {"required mapping attributes are missing in backup file", 0, ""},
	336331106:                                                     {"missing regular expression to skip tables", 0, ""},
	336331107:                                                     {"    @1SKIP_D(ATA)          skip data for table", 1, ""},
	336331108:                                                     {"regular expression to skip tables was already set", 0, ""},
	336331109:                                                     {"adjusting views dbkey length", 0, ""},
	336331110:                                                     {"updating ownership of packages, procedures and tables", 0, ""},
	336331111:                                                     {"adding missing privileges", 0, ""},
	336331112:                                                     {"adjusting the ONLINE and FORCED WRITES flags", 0, ""},
	336331113:                                                     {"    @1ST(ATISTICS) TDRW    show statistics:", 1, ""},
	336331114:                                                     {"        T                 time from start", 0, ""},
	336331115:                                                     {"        D                 delta time", 0, ""},
	336331116:                                                     {"        R                 page reads", 0, ""},
	336331117:                                                     {"        W                 page writes", 0, ""},
	Isc_gbak_missing_perf:                                         {"statistics parameter missing", 0, ""},
	Isc_gbak_wrong_perf:                                           {"wrong char \"@1\" at statistics parameter", 1, ""},
	Isc_gbak_too_long_perf:                                        {"too many chars at statistics parameter", 0, ""},
	336331121:                                                     {"total statistics", 0, ""},
	336331122:                                                     {"could not append BLOB data to batch", 0, ""},
	336331123:                                                     {"could not start batch when restoring table @1, trying old way", 1, ""},
	336331124:                                                     {"    @1KEYNAME              name of a key to be used for encryption", 1, ""},
	336331125:                                                     {"    @1CRYPT                crypt plugin name", 1, ""},
	336331126:                                                     {"    @1ZIP                  backup file is in zstd compressed format", 1, ""},
	336331127:                                                     {"Keyname parameter missing", 0, ""},
	336331128:                                                     {"Key holder parameter missing but backup file is encrypted", 0, ""},
	336331129:                                                     {"CryptPlugin parameter missing", 0, ""},
	336331130:                                                     {"Unknown crypt plugin name - use -CRYPT switch", 0, ""},
	336331131:                                                     {"Decompress error @1", 1, ""},
	336331132:                                                     {"Compress error @1", 1, ""},
	336331133:                                                     {"Key holder parameter missing", 0, ""},
	336331134:                                                     {"    @1KEYHOLDER            name of a key holder plugin", 1, ""},
	336331135:                                                     {"Decompression stream init error @1", 1, ""},
	336331136:                                                     {"Compression stream init error @1", 1, ""},
	336331137:                                                     {"Invalid reply from getInfo() when waiting for DB encryption", 0, ""},
	336331138:                                                     {"Problems with just created database encryption", 0, ""},
	336331139:                                                     {"Skipped trigger @1 on system table @2", 2, ""},
	336331140:                                                     {"    @1INCLUDE(_DATA)       backup data of table(s)", 1, ""},
	336331141:                                                     {"missing regular expression to include tables", 0, ""},
	336331142:                                                     {"regular expression to include tables was already set", 0, ""},
	336331143:                                                     {"writing database create grants", 0, ""},
	336331144:                                                     {"    database create grant for @1", 1, ""},
	336331145:                                                     {"    restoring database create grant for @1", 1, ""},
	336331146:                                                     {"restoring database create grants", 0, ""},
	336331147:                                                     {"database create grant", 0, ""},
	336331148:                                                     {"writing publications", 0, ""},
	336331149:                                                     {"    writing publication @1", 1, ""},
	336331150:                                                     {"    writing publication for table @1", 1, ""},
	336331151:                                                     {"restoring publication @1", 1, ""},
	336331152:                                                     {"publication", 0, ""},
	336331153:                                                     {"restoring publication for table @1", 1, ""},
	336331154:                                                     {"publication for table", 0, ""},
	Isc_gbak_opt_replica:                                          {"    @1REPLICA <mode>      \"none\", \"read_only\" or \"read_write\" replica mode", 1, ""},
	Isc_gbak_replica_req:                                          {"\"none\", \"read_only\" or \"read_write\" required", 0, ""},
	336331157:                                                     {"could not access batch parameters", 0, ""},
	336331158:                                                     {"    @1PAR(ALLEL)           parallel workers", 1, ""},
	Isc_gbak_missing_prl_wrks:                                     {"parallel workers parameter missing", 0, ""},
	Isc_gbak_inv_prl_wrks:                                         {"expected parallel workers, encountered \"@1\"", 1, ""},
	336331161:                                                     {"    @1D(IRECT_IO)          direct IO for backup file(s)", 1, ""},
	336331162:                                                     {"use up to @1 parallel workers", 1, ""},
	336331163:                                                     {"writing schema @1", 1, ""},
	336331164:                                                     {"writing schemas", 0, ""},
	336331165:                                                     {"restoring schema @1", 1, ""},
	336331166:                                                     {"schema", 0, ""},
	336331167:                                                     {"    @1SKIP_SCHEMA_D(ATA)    skip data for schema", 1, ""},
	336331168:                                                     {"    @1INCLUDE_SCHEMA_D(ATA) backup data of schema(s)", 1, ""},
	336331169:                                                     {"missing regular expression to skip schemas", 0, ""},
	336331170:                                                     {"missing regular expression to include schemas", 0, ""},
	336331171:                                                     {"regular expression to skip schemas was already set", 0, ""},
	336331172:                                                     {"regular expression to include schemas was already set", 0, ""},
	Isc_gbak_plugin_schema_migration:                              {"migrating @1 plugin objects to schema @2", 2, ""},
	Isc_gbak_plugin_schema_migration_err:                          {"error migrating @1 plugin objects to schema @2. Plugin objects will be in inconsistent state:", 2, ""},
	Isc_gbak_writing_constants:                                    {"writing constants", 0, "00000"},
	Isc_gbak_writing_constant:                                     {"writing constant %s", 0, "00000"},
	Isc_gbak_constant:                                             {"constant (in RDB$CONSTANTS)", 0, "00000"},
	Isc_gbak_restoring_constant:                                   {"restoring constant %s", 0, "00000"},
	336331752:                                                     {"wait for a port of feature: @1", 1, ""},
	336331753:                                                     {"    @1RDB_MAP(PING_FILE) <mapping file>        mapping file for fixing RDB2.x Java ESP UDFs on the fly", 1, ""},
	336331754:                                                     {"mapping file parameter missing", 0, ""},
	336331755:                                                     {"cannot open mapping file \"@1\"", 1, ""},
	336331756:                                                     {"Updating entry point \"@1\" -> \"@2\"", 2, ""},
	336331757:                                                     {"started at @1", 1, ""},
	336331758:                                                     {"stopped at @1", 1, ""},
	336331759:                                                     {"missing regular expression to keep tables", 0, ""},
	336331760:                                                     {"    @1KEEP_D(ATA)          keep data for table", 1, ""},
	336331761:                                                     {"regular expression to keep tables was already set", 0, ""},
	Isc_gbak_db_not_online:                                        {"Database is not online due to failure to restore one or more objects.", 0, "00000"},
	Isc_gbak_gfix_online:                                          {"Run gfix -online to bring database online.", 0, "00000"},
	336331764:                                                     {"    @1IG(NORE)             backup: ignore bad checksums\n\t\t\t  restore: ignore errors in object's BLR", 1, ""},
	336331765:                                                     {"    @1PAR(ALLEL)           parallel workers", 1, ""},
	336331766:                                                     {"parallel workers parameter missing", 0, ""},
	336331767:                                                     {"expected parallel workers, encountered \"@1\"", 1, ""},
	336331768:                                                     {"Warning: The \"@1\" feature is not supported", 1, ""},
	336331777:                                                     {"parameter for option -@1 is missing", 1, ""},
	Isc_gbak_db_not_online_reason:                                 {"Database is not online due to @1.", 1, "00000"},
	336331779:                                                     {"Compressor plugin '@1' not found", 1, ""},
	336331780:                                                     {"    @1COM(PRESSOR)         plugin name, compression level and threads count", 1, ""},
	336331781:                                                     {"writing foreign servers", 0, ""},
	336331782:                                                     {"    writing foreign server @1", 1, ""},
	336331783:                                                     {"restoring foreign server @1", 1, ""},
	336331784:                                                     {"foreign server", 0, ""},
	336331785:                                                     {"         writing foreign server option @1", 1, ""},
	336331786:                                                     {"    restoring foreign server option @1", 1, ""},
	336331787:                                                     {"foreign server option", 0, ""},
	336331788:                                                     {"         writing foreign table option @1", 1, ""},
	336331789:                                                     {"    restoring foreign table option @1", 1, ""},
	336331790:                                                     {"foreign table option", 0, ""},
	336331791:                                                     {"              writing foreign table column option @1", 1, ""},
	336331792:                                                     {"         restoring foreign table column option @1", 1, ""},
	336331793:                                                     {"foreign table column option", 0, ""},
	336331794:                                                     {"writing foreign user mappings", 0, ""},
	336331795:                                                     {"    writing foreign user mapping for user @1 and server @2", 2, ""},
	336331796:                                                     {"restoring foreign user mapping for user @1 and server @2", 2, ""},
	336331797:                                                     {"foreign user mapping", 0, ""},
	336331798:                                                     {"         writing foreign user mapping option @1", 1, ""},
	336331799:                                                     {"    restoring foreign user mapping option @1", 1, ""},
	336331800:                                                     {"foreign user mapping option", 0, ""},
	336331801:                                                     {"Expected @1 instead of @2", 2, ""},
	336331802:                                                     {"compression level @1", 1, ""},
	336331803:                                                     {"'@1' compression plugin used", 1, ""},
	336331804:                                                     {"use up to @1 compression threads", 1, ""},
	336396289:                                                     {"Red Database error", 0, ""},
	336396362:                                                     {"Rollback not performed", 0, ""},
	336396364:                                                     {"Connection error", 0, ""},
	336396365:                                                     {"Connection not established", 0, ""},
	336396366:                                                     {"Connection authorization failure.", 0, ""},
	336396375:                                                     {"deadlock", 0, ""},
	336396376:                                                     {"Unsuccessful execution caused by deadlock.", 0, ""},
	336396377:                                                     {"record from transaction @1 is stuck in limbo", 1, ""},
	336396379:                                                     {"operation completed with errors", 0, ""},
	336396382:                                                     {"the SQL statement cannot be executed", 0, ""},
	336396384:                                                     {"Unsuccessful execution caused by an unavailable resource.", 0, ""},
	336396386:                                                     {"Unsuccessful execution caused by a system error that precludes successful execution of subsequent statements", 0, ""},
	336396387:                                                     {"Unsuccessful execution caused by system error that does not preclude successful execution of subsequent statements", 0, ""},
	336396446:                                                     {"Wrong numeric type", 0, ""},
	336396447:                                                     {"too many versions", 0, ""},
	336396448:                                                     {"intermediate journal file full", 0, ""},
	336396449:                                                     {"journal file wrong format", 0, ""},
	336396450:                                                     {"database @1 shutdown in @2 seconds", 2, ""},
	336396451:                                                     {"restart shared cache manager", 0, ""},
	336396452:                                                     {"exception @1", 1, ""},
	336396453:                                                     {"bad checksum", 0, ""},
	336396454:                                                     {"refresh range number @1 not found", 1, ""},
	336396455:                                                     {"expression evaluation not supported", 0, ""},
	336396456:                                                     {"FOREIGN KEY column count does not match PRIMARY KEY", 0, ""},
	336396457:                                                     {"Attempt to define a second PRIMARY KEY for the same table", 0, ""},
	336396458:                                                     {"column used with aggregate", 0, ""},
	336396459:                                                     {"invalid column reference", 0, ""},
	336396460:                                                     {"invalid key position", 0, ""},
	336396461:                                                     {"invalid direction for find operation", 0, ""},
	336396462:                                                     {"Invalid statement handle", 0, ""},
	336396463:                                                     {"invalid lock handle", 0, ""},
	336396464:                                                     {"invalid lock level @1", 1, ""},
	336396465:                                                     {"invalid bookmark handle", 0, ""},
	336396468:                                                     {"wrong or obsolete version", 0, ""},
	336396471:                                                     {"The INSERT, UPDATE, DELETE, DDL or authorization statement cannot be executed because the transaction is inquiry only", 0, ""},
	336396472:                                                     {"external file could not be opened for output", 0, ""},
	336396477:                                                     {"multiple rows in singleton select", 0, ""},
	336396478:                                                     {"No subqueries permitted for VIEW WITH CHECK OPTION", 0, ""},
	336396479:                                                     {"DISTINCT, GROUP or HAVING not permitted for VIEW WITH CHECK OPTION", 0, ""},
	336396480:                                                     {"Only one table allowed for VIEW WITH CHECK OPTION", 0, ""},
	336396481:                                                     {"No WHERE clause for VIEW WITH CHECK OPTION", 0, ""},
	336396482:                                                     {"Only simple column names permitted for VIEW WITH CHECK OPTION", 0, ""},
	336396484:                                                     {"An error was found in the application program input parameters for the SQL statement.", 0, ""},
	336396485:                                                     {"Invalid insert or update value(s): object columns are constrained - no 2 table rows can have duplicate column values", 0, ""},
	336396486:                                                     {"Arithmetic overflow or division by zero has occurred.", 0, ""},
	336396594:                                                     {"cannot access column @1 in view @2", 2, ""},
	336396595:                                                     {"Too many concurrent executions of the same request", 0, ""},
	336396596:                                                     {"maximum indexes per table (@1) exceeded", 1, ""},
	336396597:                                                     {"new record size of @1 bytes is too big", 1, ""},
	336396598:                                                     {"segments not allowed in expression index @1", 1, ""},
	336396599:                                                     {"wrong page type", 0, ""},
	336396603:                                                     {"invalid ARRAY or BLOB operation", 0, ""},
	336396611:                                                     {"@1 extension error", 1, ""},
	336396624:                                                     {"key size exceeds implementation restriction for index \"@1\"", 1, ""},
	336396625:                                                     {"definition error for index @1", 1, ""},
	336396628:                                                     {"cannot create index", 0, ""},
	336396651:                                                     {"duplicate specification of @1 - not supported", 1, ""},
	336396663:                                                     {"The insert failed because a column definition includes validation constraints.", 0, ""},
	336396670:                                                     {"Cannot delete object referenced by another object", 0, ""},
	336396671:                                                     {"Cannot modify object referenced by another object", 0, ""},
	336396672:                                                     {"Object is referenced by another object", 0, ""},
	336396673:                                                     {"lock on conflicts with existing lock", 0, ""},
	336396681:                                                     {"This operation is not defined for system tables.", 0, ""},
	336396683:                                                     {"Inappropriate self-reference of column", 0, ""},
	336396684:                                                     {"Illegal array dimension range", 0, ""},
	336396687:                                                     {"database or file exists", 0, ""},
	336396688:                                                     {"sort error: corruption in data structure", 0, ""},
	336396689:                                                     {"node not supported", 0, ""},
	336396690:                                                     {"Shadow number must be a positive integer", 0, ""},
	336396691:                                                     {"Preceding file did not specify length, so @1 must include starting page number", 1, ""},
	336396692:                                                     {"illegal operation when at beginning of stream", 0, ""},
	336396693:                                                     {"the current position is on a crack", 0, ""},
	336396735:                                                     {"cannot modify an existing user privilege", 0, ""},
	336396736:                                                     {"user does not have the privilege to perform operation", 0, ""},
	336396737:                                                     {"This user does not have privilege to perform this operation on this object.", 0, ""},
	336396756:                                                     {"transaction marked invalid by I/O error", 0, ""},
	336396757:                                                     {"Cannot prepare a CREATE DATABASE/SCHEMA statement", 0, ""},
	336396758:                                                     {"violation of FOREIGN KEY constraint \"@1\"", 1, ""},
	336396769:                                                     {"The prepare statement identifies a prepare statement with an open cursor", 0, ""},
	336396770:                                                     {"Unknown statement or request", 0, ""},
	336396778:                                                     {"Attempt to update non-updatable cursor", 0, ""},
	336396780:                                                     {"The cursor identified in the UPDATE or DELETE statement is not positioned on a row.", 0, ""},
	336396784:                                                     {"Unknown cursor", 0, ""},
	336396786:                                                     {"The cursor identified in an OPEN statement is already open.", 0, ""},
	336396787:                                                     {"The cursor identified in a FETCH or CLOSE statement is not open.", 0, ""},
	336396875:                                                     {"Overflow occurred during data type conversion.", 0, ""},
	336396881:                                                     {"null segment of UNIQUE KEY", 0, ""},
	336396882:                                                     {"subscript out of bounds", 0, ""},
	336396886:                                                     {"data operation not supported", 0, ""},
	336396887:                                                     {"invalid comparison operator for find operation", 0, ""},
	336396974:                                                     {"Cannot transliterate character between character sets", 0, ""},
	336396975:                                                     {"count of column list and variable list do not match", 0, ""},
	336396985:                                                     {"Incompatible column/host variable data type", 0, ""},
	336396991:                                                     {"Operation violates CHECK constraint @1 on view or table", 1, ""},
	336396992:                                                     {"internal Red Database consistency check (invalid RDB$CONSTRAINT_TYPE)", 0, ""},
	336396993:                                                     {"Cannot update constraints (RDB$RELATION_CONSTRAINTS).", 0, ""},
	336396994:                                                     {"Cannot delete CHECK constraint entry (RDB$CHECK_CONSTRAINTS)", 0, ""},
	336396995:                                                     {"Cannot update constraints (RDB$CHECK_CONSTRAINTS).", 0, ""},
	336396996:                                                     {"Cannot update constraints (RDB$REF_CONSTRAINTS).", 0, ""},
	336396997:                                                     {"Column used in a PRIMARY constraint must be NOT NULL.", 0, ""},
	336397004:                                                     {"index @1 cannot be used in the specified plan", 1, ""},
	336397005:                                                     {"table @1 is referenced in the plan but not the from list", 1, ""},
	336397006:                                                     {"the table @1 is referenced twice; use aliases to differentiate", 1, ""},
	336397007:                                                     {"table @1 is not referenced in plan", 1, ""},
	336397027:                                                     {"Log file specification partition error", 0, ""},
	336397028:                                                     {"Cache or Log redefined", 0, ""},
	336397029:                                                     {"Write-ahead Log with shadowing configuration not allowed", 0, ""},
	336397030:                                                     {"Overflow log specification required for round-robin log", 0, ""},
	336397031:                                                     {"WAL defined; Cache Manager must be started first", 0, ""},
	336397033:                                                     {"Write-ahead Log without shared cache configuration not allowed", 0, ""},
	336397034:                                                     {"Cannot start WAL writer for the database @1", 1, ""},
	336397035:                                                     {"WAL writer synchronization error for the database @1", 1, ""},
	336397036:                                                     {"WAL setup error.  Please see Red Database log.", 0, ""},
	336397037:                                                     {"WAL buffers cannot be increased.  Please see Red Database log.", 0, ""},
	336397038:                                                     {"WAL writer - Journal server communication error.  Please see Red Database log.", 0, ""},
	336397039:                                                     {"WAL I/O error.  Please see Red Database log.", 0, ""},
	336397040:                                                     {"Unable to roll over; please see Red Database log.", 0, ""},
	336397041:                                                     {"obsolete", 0, ""},
	336397042:                                                     {"obsolete", 0, ""},
	336397043:                                                     {"obsolete", 0, ""},
	336397044:                                                     {"obsolete", 0, ""},
	336397045:                                                     {"database does not use Write-ahead Log", 0, ""},
	336397046:                                                     {"Cannot roll over to the next log file @1", 1, ""},
	336397047:                                                     {"obsolete", 0, ""},
	336397048:                                                     {"obsolete", 0, ""},
	336397049:                                                     {"Cache or Log size too small", 0, ""},
	336397050:                                                     {"Log record header too small at offset @1 in log file @2", 2, ""},
	336397051:                                                     {"Incomplete log record at offset @1 in log file @2", 2, ""},
	336397052:                                                     {"Unexpected end of log file @1 at offset @2", 2, ""},
	336397053:                                                     {"Database name in the log file @1 is different", 1, ""},
	336397054:                                                     {"Log file @1 not closed properly; database recovery may be required", 1, ""},
	336397055:                                                     {"Log file @1 not latest in the chain but open flag still set", 1, ""},
	336397056:                                                     {"Invalid version of log file @1", 1, ""},
	336397057:                                                     {"Log file header of @1 too small", 1, ""},
	336397058:                                                     {"obsolete", 0, ""},
	336397069:                                                     {"table @1 is not defined", 1, ""},
	336397080:                                                     {"invalid ORDER BY clause", 0, ""},
	336397082:                                                     {"Column does not belong to referenced table", 0, ""},
	336397083:                                                     {"column @1 is not defined in table @2", 2, ""},
	336397084:                                                     {"Undefined name", 0, ""},
	336397085:                                                     {"Ambiguous column reference.", 0, ""},
	336397116:                                                     {"function @1 is not defined", 1, ""},
	336397117:                                                     {"Invalid data type, length, or value", 0, ""},
	336397118:                                                     {"Invalid number of arguments", 0, ""},
	336397126:                                                     {"dbkey not available for multi-table views", 0, ""},
	336397130:                                                     {"number of columns does not match select list", 0, ""},
	336397131:                                                     {"must specify column name for view select expression", 0, ""},
	336397133:                                                     {"@1 is not a valid base table of the specified view", 1, ""},
	336397137:                                                     {"This column cannot be updated because it is derived from an SQL function or expression.", 0, ""},
	336397138:                                                     {"The object of the INSERT, DELETE or UPDATE statement is a view for which the requested operation is not permitted.", 0, ""},
	336397183:                                                     {"Invalid String", 0, ""},
	336397184:                                                     {"Invalid token", 0, ""},
	336397185:                                                     {"Invalid numeric literal", 0, ""},
	336397203:                                                     {"An error occurred while trying to update the security database", 0, ""},
	336397204:                                                     {"non-SQL security class defined", 0, ""},
	Isc_dsql_too_old_ods:                                          {"ODS versions before ODS@1 are not supported", 1, "HY000"},
	Isc_dsql_table_not_found:                                      {"Table @1 does not exist", 1, "42S02"},
	Isc_dsql_view_not_found:                                       {"View @1 does not exist", 1, "42S02"},
	Isc_dsql_line_col_error:                                       {"At line @1, column @2", 2, "42000"},
	Isc_dsql_unknown_pos:                                          {"At unknown line and column", 0, "42000"},
	Isc_dsql_no_dup_name:                                          {"Column @1 cannot be repeated in @2 statement", 2, "HY000"},
	Isc_dsql_too_many_values:                                      {"Too many values (more than @1) in member list to match against", 1, "07002"},
	Isc_dsql_no_array_computed:                                    {"Array and BLOB data types not allowed in computed field", 0, "HY000"},
	Isc_dsql_implicit_domain_name:                                 {"Implicit domain name @1 not allowed in user created domain", 1, "00000"},
	Isc_dsql_only_can_subscript_array:                             {"scalar operator used on field @1 which is not an array", 1, "42000"},
	Isc_dsql_max_sort_items:                                       {"cannot sort on more than 255 items", 0, "54011"},
	Isc_dsql_max_group_items:                                      {"cannot group on more than 255 items", 0, "54011"},
	Isc_dsql_conflicting_sort_field:                               {"Cannot include the same field (@1.@2) twice in the ORDER BY clause with conflicting sorting options", 2, "42000"},
	Isc_dsql_derived_table_more_columns:                           {"column list from derived table @1 has more columns than the number of items in its SELECT statement", 1, "54001"},
	Isc_dsql_derived_table_less_columns:                           {"column list from derived table @1 has less columns than the number of items in its SELECT statement", 1, "54001"},
	Isc_dsql_derived_field_unnamed:                                {"no column name specified for column number @1 in derived table @2", 2, "42000"},
	Isc_dsql_derived_field_dup_name:                               {"column @1 was specified multiple times for derived table @2", 2, "42000"},
	Isc_dsql_derived_alias_select:                                 {"Internal dsql error: alias type expected by pass1_expand_select_node", 0, "42000"},
	Isc_dsql_derived_alias_field:                                  {"Internal dsql error: alias type expected by pass1_field", 0, "42000"},
	Isc_dsql_auto_field_bad_pos:                                   {"Internal dsql error: column position out of range in pass1_union_auto_cast", 0, "42000"},
	Isc_dsql_cte_wrong_reference:                                  {"Recursive CTE member (@1) can refer itself only in FROM clause", 1, "42000"},
	Isc_dsql_cte_cycle:                                            {"CTE '@1' has cyclic dependencies", 1, "42000"},
	Isc_dsql_cte_outer_join:                                       {"Recursive member of CTE can't be member of an outer join", 0, "42000"},
	Isc_dsql_cte_mult_references:                                  {"Recursive member of CTE can't reference itself more than once", 0, "42000"},
	Isc_dsql_cte_not_a_union:                                      {"Recursive CTE (@1) must be an UNION", 1, "42000"},
	Isc_dsql_cte_nonrecurs_after_recurs:                           {"CTE '@1' defined non-recursive member after recursive", 1, "42000"},
	Isc_dsql_cte_wrong_clause:                                     {"Recursive member of CTE '@1' has @2 clause", 2, "42000"},
	Isc_dsql_cte_union_all:                                        {"Recursive members of CTE (@1) must be linked with another members via UNION ALL", 1, "42000"},
	Isc_dsql_cte_miss_nonrecursive:                                {"Non-recursive member is missing in CTE '@1'", 1, "42000"},
	Isc_dsql_cte_nested_with:                                      {"WITH clause can't be nested", 0, "42000"},
	Isc_dsql_col_more_than_once_using:                             {"column @1 appears more than once in USING clause", 1, "42000"},
	Isc_dsql_unsupp_feature_dialect:                               {"feature is not supported in dialect @1", 1, "HY000"},
	Isc_dsql_cte_not_used:                                         {"CTE \"@1\" is not used in query", 1, "42000"},
	Isc_dsql_col_more_than_once_view:                              {"column @1 appears more than once in ALTER VIEW", 1, "42000"},
	Isc_dsql_unsupported_in_auto_trans:                            {"@1 is not supported inside IN AUTONOMOUS TRANSACTION block", 1, "HY000"},
	Isc_dsql_eval_unknode:                                         {"Unknown node type @1 in dsql/GEN_expr", 1, "42000"},
	Isc_dsql_agg_wrongarg:                                         {"Argument for @1 in dialect 1 must be string or numeric", 1, "42000"},
	Isc_dsql_agg2_wrongarg:                                        {"Argument for @1 in dialect 3 must be numeric", 1, "42000"},
	Isc_dsql_nodateortime_pm_string:                               {"Strings cannot be added to or subtracted from DATE or TIME types", 0, "42000"},
	Isc_dsql_invalid_datetime_subtract:                            {"Invalid data type for subtraction involving DATE, TIME or TIMESTAMP types", 0, "42000"},
	Isc_dsql_invalid_dateortime_add:                               {"Adding two DATE values or two TIME values is not allowed", 0, "42000"},
	Isc_dsql_invalid_type_minus_date:                              {"DATE value cannot be subtracted from the provided data type", 0, "42000"},
	Isc_dsql_nostring_addsub_dial3:                                {"Strings cannot be added or subtracted in dialect 3", 0, "42000"},
	Isc_dsql_invalid_type_addsub_dial3:                            {"Invalid data type for addition or subtraction in dialect 3", 0, "42000"},
	Isc_dsql_invalid_type_multip_dial1:                            {"Invalid data type for multiplication in dialect 1", 0, "42000"},
	Isc_dsql_nostring_multip_dial3:                                {"Strings cannot be multiplied in dialect 3", 0, "42000"},
	Isc_dsql_invalid_type_multip_dial3:                            {"Invalid data type for multiplication in dialect 3", 0, "42000"},
	Isc_dsql_mustuse_numeric_div_dial1:                            {"Division in dialect 1 must be between numeric data types", 0, "42000"},
	Isc_dsql_nostring_div_dial3:                                   {"Strings cannot be divided in dialect 3", 0, "42000"},
	Isc_dsql_invalid_type_div_dial3:                               {"Invalid data type for division in dialect 3", 0, "42000"},
	Isc_dsql_nostring_neg_dial3:                                   {"Strings cannot be negated (applied the minus operator) in dialect 3", 0, "42000"},
	Isc_dsql_invalid_type_neg:                                     {"Invalid data type for negation (minus operator)", 0, "42000"},
	Isc_dsql_max_distinct_items:                                   {"Cannot have more than 255 items in DISTINCT / UNION DISTINCT list", 0, "54011"},
	Isc_dsql_alter_charset_failed:                                 {"ALTER CHARACTER SET @1 failed", 1, "42000"},
	Isc_dsql_comment_on_failed:                                    {"COMMENT ON @1 failed", 1, "42000"},
	Isc_dsql_create_func_failed:                                   {"CREATE FUNCTION @1 failed", 1, "42000"},
	Isc_dsql_alter_func_failed:                                    {"ALTER FUNCTION @1 failed", 1, "42000"},
	Isc_dsql_create_alter_func_failed:                             {"CREATE OR ALTER FUNCTION @1 failed", 1, "42000"},
	Isc_dsql_drop_func_failed:                                     {"DROP FUNCTION @1 failed", 1, "42000"},
	Isc_dsql_recreate_func_failed:                                 {"RECREATE FUNCTION @1 failed", 1, "42000"},
	Isc_dsql_create_proc_failed:                                   {"CREATE PROCEDURE @1 failed", 1, "42000"},
	Isc_dsql_alter_proc_failed:                                    {"ALTER PROCEDURE @1 failed", 1, "42000"},
	Isc_dsql_create_alter_proc_failed:                             {"CREATE OR ALTER PROCEDURE @1 failed", 1, "42000"},
	Isc_dsql_drop_proc_failed:                                     {"DROP PROCEDURE @1 failed", 1, "42000"},
	Isc_dsql_recreate_proc_failed:                                 {"RECREATE PROCEDURE @1 failed", 1, "42000"},
	Isc_dsql_create_trigger_failed:                                {"CREATE TRIGGER @1 failed", 1, "42000"},
	Isc_dsql_alter_trigger_failed:                                 {"ALTER TRIGGER @1 failed", 1, "42000"},
	Isc_dsql_create_alter_trigger_failed:                          {"CREATE OR ALTER TRIGGER @1 failed", 1, "42000"},
	Isc_dsql_drop_trigger_failed:                                  {"DROP TRIGGER @1 failed", 1, "42000"},
	Isc_dsql_recreate_trigger_failed:                              {"RECREATE TRIGGER @1 failed", 1, "42000"},
	Isc_dsql_create_collation_failed:                              {"CREATE COLLATION @1 failed", 1, "42000"},
	Isc_dsql_drop_collation_failed:                                {"DROP COLLATION @1 failed", 1, "42000"},
	Isc_dsql_create_domain_failed:                                 {"CREATE DOMAIN @1 failed", 1, "42000"},
	Isc_dsql_alter_domain_failed:                                  {"ALTER DOMAIN @1 failed", 1, "42000"},
	Isc_dsql_drop_domain_failed:                                   {"DROP DOMAIN @1 failed", 1, "42000"},
	Isc_dsql_create_except_failed:                                 {"CREATE EXCEPTION @1 failed", 1, "42000"},
	Isc_dsql_alter_except_failed:                                  {"ALTER EXCEPTION @1 failed", 1, "42000"},
	Isc_dsql_create_alter_except_failed:                           {"CREATE OR ALTER EXCEPTION @1 failed", 1, "42000"},
	Isc_dsql_recreate_except_failed:                               {"RECREATE EXCEPTION @1 failed", 1, "42000"},
	Isc_dsql_drop_except_failed:                                   {"DROP EXCEPTION @1 failed", 1, "42000"},
	Isc_dsql_create_sequence_failed:                               {"CREATE SEQUENCE @1 failed", 1, "42000"},
	Isc_dsql_create_table_failed:                                  {"CREATE TABLE @1 failed", 1, "42000"},
	Isc_dsql_alter_table_failed:                                   {"ALTER TABLE @1 failed", 1, "42000"},
	Isc_dsql_drop_table_failed:                                    {"DROP TABLE @1 failed", 1, "42000"},
	Isc_dsql_recreate_table_failed:                                {"RECREATE TABLE @1 failed", 1, "42000"},
	Isc_dsql_create_pack_failed:                                   {"CREATE PACKAGE @1 failed", 1, "42000"},
	Isc_dsql_alter_pack_failed:                                    {"ALTER PACKAGE @1 failed", 1, "42000"},
	Isc_dsql_create_alter_pack_failed:                             {"CREATE OR ALTER PACKAGE @1 failed", 1, "42000"},
	Isc_dsql_drop_pack_failed:                                     {"DROP PACKAGE @1 failed", 1, "42000"},
	Isc_dsql_recreate_pack_failed:                                 {"RECREATE PACKAGE @1 failed", 1, "42000"},
	Isc_dsql_create_pack_body_failed:                              {"CREATE PACKAGE BODY @1 failed", 1, "42000"},
	Isc_dsql_drop_pack_body_failed:                                {"DROP PACKAGE BODY @1 failed", 1, "42000"},
	Isc_dsql_recreate_pack_body_failed:                            {"RECREATE PACKAGE BODY @1 failed", 1, "42000"},
	Isc_dsql_create_view_failed:                                   {"CREATE VIEW @1 failed", 1, "42000"},
	Isc_dsql_alter_view_failed:                                    {"ALTER VIEW @1 failed", 1, "42000"},
	Isc_dsql_create_alter_view_failed:                             {"CREATE OR ALTER VIEW @1 failed", 1, "42000"},
	Isc_dsql_recreate_view_failed:                                 {"RECREATE VIEW @1 failed", 1, "42000"},
	Isc_dsql_drop_view_failed:                                     {"DROP VIEW @1 failed", 1, "42000"},
	Isc_dsql_drop_sequence_failed:                                 {"DROP SEQUENCE @1 failed", 1, "42000"},
	Isc_dsql_recreate_sequence_failed:                             {"RECREATE SEQUENCE @1 failed", 1, "42000"},
	Isc_dsql_drop_index_failed:                                    {"DROP INDEX @1 failed", 1, "42000"},
	Isc_dsql_drop_filter_failed:                                   {"DROP FILTER @1 failed", 1, "42000"},
	Isc_dsql_drop_shadow_failed:                                   {"DROP SHADOW @1 failed", 1, "42000"},
	Isc_dsql_drop_role_failed:                                     {"DROP ROLE @1 failed", 1, "42000"},
	Isc_dsql_drop_user_failed:                                     {"DROP USER @1 failed", 1, "42000"},
	Isc_dsql_create_role_failed:                                   {"CREATE ROLE @1 failed", 1, "42000"},
	Isc_dsql_alter_role_failed:                                    {"ALTER ROLE @1 failed", 1, "42000"},
	Isc_dsql_alter_index_failed:                                   {"ALTER INDEX @1 failed", 1, "42000"},
	Isc_dsql_alter_database_failed:                                {"ALTER DATABASE failed", 0, "42000"},
	Isc_dsql_create_shadow_failed:                                 {"CREATE SHADOW @1 failed", 1, "42000"},
	Isc_dsql_create_filter_failed:                                 {"DECLARE FILTER @1 failed", 1, "42000"},
	Isc_dsql_create_index_failed:                                  {"CREATE INDEX @1 failed", 1, "42000"},
	Isc_dsql_create_user_failed:                                   {"CREATE USER @1 failed", 1, "42000"},
	Isc_dsql_alter_user_failed:                                    {"ALTER USER @1 failed", 1, "42000"},
	Isc_dsql_grant_failed:                                         {"GRANT failed", 0, "42000"},
	Isc_dsql_revoke_failed:                                        {"REVOKE failed", 0, "42000"},
	Isc_dsql_cte_recursive_aggregate:                              {"Recursive member of CTE cannot use aggregate or window function", 0, "42000"},
	Isc_dsql_mapping_failed:                                       {"@2 MAPPING @1 failed", 2, "42000"},
	Isc_dsql_alter_sequence_failed:                                {"ALTER SEQUENCE @1 failed", 1, "42000"},
	Isc_dsql_create_generator_failed:                              {"CREATE GENERATOR @1 failed", 1, "42000"},
	Isc_dsql_set_generator_failed:                                 {"SET GENERATOR @1 failed", 1, "42000"},
	Isc_dsql_wlock_simple:                                         {"WITH LOCK can be used only with a single physical table", 0, "42000"},
	Isc_dsql_firstskip_rows:                                       {"FIRST/SKIP cannot be used with OFFSET/FETCH or ROWS", 0, "42000"},
	Isc_dsql_wlock_aggregates:                                     {"WITH LOCK cannot be used with aggregates", 0, "42000"},
	Isc_dsql_wlock_conflict:                                       {"WITH LOCK cannot be used with @1", 1, "42000"},
	Isc_dsql_max_exception_arguments:                              {"Number of arguments (@1) exceeds the maximum (@2) number of EXCEPTION USING arguments", 2, "07002"},
	Isc_dsql_string_byte_length:                                   {"String literal with @1 bytes exceeds the maximum length of @2 bytes", 2, "42000"},
	Isc_dsql_string_char_length:                                   {"String literal with @1 characters exceeds the maximum length of @2 characters for the @3 character set", 3, "42000"},
	Isc_dsql_max_nesting:                                          {"Too many BEGIN...END nesting. Maximum level is @1", 1, "07002"},
	Isc_dsql_recreate_user_failed:                                 {"RECREATE USER @1 failed", 1, "42000"},
	Isc_dsql_table_value_many_columns:                             {"the number of fields exceeds the limit for the @1 operator. Expected @2, received @3", 3, "54001"},
	Isc_dsql_create_schema_failed:                                 {"CREATE SCHEMA @1 failed", 1, "42000"},
	Isc_dsql_drop_schema_failed:                                   {"DROP SCHEMA @1 failed", 1, "42000"},
	Isc_dsql_recreate_schema_failed:                               {"RECREATE SCHEMA @1 failed", 1, "42000"},
	Isc_dsql_alter_schema_failed:                                  {"ALTER SCHEMA @1 failed", 1, "42000"},
	Isc_dsql_create_alter_schema_failed:                           {"CREATE OR ALTER SCHEMA @1 failed", 1, "42000"},
	Isc_dsql_create_const_failed:                                  {"CREATE CONSTANT @1 failed", 1, "42000"},
	Isc_dsql_alter_const_failed:                                   {"ALTER CONSTANT @1 failed", 1, "42000"},
	Isc_dsql_create_alter_const_failed:                            {"CREATE OR ALTER CONSTANT @1 failed", 1, "42000"},
	Isc_dsql_create_policy_failed:                                 {"CREATE POLICY @1 failed", 1, "42000"},
	Isc_dsql_alter_policy_failed:                                  {"ALTER POLICY @1 failed", 1, "42000"},
	Isc_dsql_drop_policy_failed:                                   {"DROP POLICY @1 failed", 1, "42000"},
	Isc_dsql_agg_wrongtype:                                        {"Argument for @1 must be field node", 1, "42000"},
	Isc_dsql_create_job_failed:                                    {"CREATE JOB @1 failed", 1, "42000"},
	Isc_dsql_alter_job_failed:                                     {"ALTER JOB @1 failed", 1, "42000"},
	Isc_dsql_create_alter_job_failed:                              {"CREATE OR ALTER JOB @1 failed", 1, "42000"},
	Isc_dsql_drop_job_failed:                                      {"DROP JOB @1 failed", 1, "42000"},
	Isc_dsql_recreate_job_failed:                                  {"RECREATE JOB @1 failed", 1, "42000"},
	Isc_dsql_create_foreign_server_failed:                         {"CREATE SERVER @1 failed", 1, "42000"},
	Isc_dsql_alter_foreign_server_failed:                          {"ALTER SERVER @1 failed", 1, "42000"},
	Isc_dsql_create_alter_foreign_server_failed: {"CREATE OR ALTER SERVER @1 failed", 1, "42000"},
	Isc_dsql_recreate_foreign_server_failed:     {"RECREATE SERVER @1 failed", 1, "42000"},
	Isc_dsql_drop_foreign_server_failed:         {"DROP SERVER @1 failed", 1, "42000"},
	Isc_dsql_create_user_mapping_failed:         {"CREATE USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_alter_user_mapping_failed:          {"ALTER USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_create_alter_user_mapping_failed:   {"CREATE OR ALTER USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_recreate_user_mapping_failed:       {"RECREATE USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_drop_user_mapping_failed:           {"DROP USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_comment_on_user_mapping_failed:     {"COMMENT ON USER MAPPING FOR USER @1 AND SERVER @2 failed", 2, "42000"},
	Isc_dsql_foreign_table_not_found:            {"Foreign table @1 does not exist", 1, "42S02"},
	Isc_dsql_create_foreign_table_failed:        {"CREATE FOREIGN TABLE @1 failed", 1, "42000"},
	Isc_dsql_alter_foreign_table_failed:         {"ALTER FOREIGN TABLE @1 failed", 1, "42000"},
	Isc_dsql_drop_foreign_table_failed:          {"DROP FOREIGN TABLE @1 failed", 1, "42000"},
	Isc_dsql_recreate_foreign_table_failed:      {"RECREATE FOREIGN TABLE @1 failed", 1, "42000"},
	336461924:                                   {"Row not found for fetch, update or delete, or the result of a query is an empty table.", 0, ""},
	336461925:                                   {"segment buffer length shorter than expected", 0, ""},
	336462125:                                   {"Datatype needs modification", 0, ""},
	336462436:                                   {"Duplicate column or domain name found.", 0, ""},
	336527507:                                   {"invalid block type encountered", 0, ""},
	336527508:                                   {"wrong packet type", 0, ""},
	336527509:                                   {"cannot map page", 0, ""},
	336527510:                                   {"request to allocate invalid block type", 0, ""},
	336527511:                                   {"request to allocate block type larger than maximum size", 0, ""},
	336527512:                                   {"memory pool free list is invalid", 0, ""},
	336527513:                                   {"invalid pool id encountered", 0, ""},
	336527514:                                   {"attempt to release free block", 0, ""},
	336527515:                                   {"attempt to release block overlapping following free block", 0, ""},
	336527516:                                   {"attempt to release block overlapping prior free block", 0, ""},
	336527517:                                   {"cannot sort on a field that does not exist", 0, ""},
	336527518:                                   {"database file not available", 0, ""},
	336527519:                                   {"cannot assert logical lock", 0, ""},
	336527520:                                   {"wrong ACL version", 0, ""},
	336527521:                                   {"shadow block not found", 0, ""},
	336527522:                                   {"shadow lock not synchronized properly", 0, ""},
	336527523:                                   {"root file name not listed for shadow", 0, ""},
	336527524:                                   {"failed to remove symbol from hash table", 0, ""},
	336527525:                                   {"cannot find tip page", 0, ""},
	336527526:                                   {"invalid rsb type", 0, ""},
	336527527:                                   {"invalid SEND request", 0, ""},
	336527528:                                   {"looper: action not yet implemented", 0, ""},
	336527529:                                   {"return data type not supported", 0, ""},
	336527530:                                   {"unexpected reply from journal server", 0, ""},
	336527531:                                   {"journal server is incompatible version", 0, ""},
	336527532:                                   {"journal server refused connection", 0, ""},
	336527533:                                   {"referenced index description not found", 0, ""},
	336527534:                                   {"index key too big", 0, ""},
	336527535:                                   {"partner index description not found", 0, ""},
	336527536:                                   {"bad difference record", 0, ""},
	336527537:                                   {"applied differences will not fit in record", 0, ""},
	336527538:                                   {"record length inconsistent", 0, ""},
	336527539:                                   {"decompression overran buffer", 0, ""},
	336527540:                                   {"cannot reposition for update after sort for RMS", 0, ""},
	336527541:                                   {"external access type not implemented", 0, ""},
	336527542:                                   {"differences record too long", 0, ""},
	336527543:                                   {"wrong record length", 0, ""},
	336527544:                                   {"limbo impossible", 0, ""},
	336527545:                                   {"wrong record version", 0, ""},
	336527546:                                   {"record disappeared", 0, ""},
	336527547:                                   {"cannot delete system tables", 0, ""},
	336527548:                                   {"cannot update erased record", 0, ""},
	336527549:                                   {"comparison not supported for specified data types", 0, ""},
	336527550:                                   {"conversion not supported for specified data types", 0, ""},
	336527551:                                   {"conversion error", 0, ""},
	336527552:                                   {"overflow during conversion", 0, ""},
	336527553:                                   {"null or invalid array", 0, ""},
	336527554:                                   {"BLOB not found", 0, ""},
	336527555:                                   {"cannot update old BLOB", 0, ""},
	336527556:                                   {"relation for array not known", 0, ""},
	336527557:                                   {"field for array not known", 0, ""},
	336527558:                                   {"array subscript computation error", 0, ""},
	336527559:                                   {"expected field node", 0, ""},
	336527560:                                   {"invalid BLOB ID", 0, ""},
	336527561:                                   {"cannot find BLOB page", 0, ""},
	336527562:                                   {"unknown data type", 0, ""},
	336527563:                                   {"shadow block not found for extend file", 0, ""},
	336527564:                                   {"index inconsistent", 0, ""},
	336527565:                                   {"index bucket overfilled", 0, ""},
	336527566:                                   {"exceeded index level", 0, ""},
	336527567:                                   {"page already in use", 0, ""},
	336527568:                                   {"page not accessed for write", 0, ""},
	336527569:                                   {"attempt to release page not acquired", 0, ""},
	336527570:                                   {"page in use during flush", 0, ""},
	336527571:                                   {"attempt to remove page from dirty page list when not there", 0, ""},
	336527572:                                   {"CCH_precedence: block marked", 0, ""},
	336527573:                                   {"insufficient cache size", 0, ""},
	336527574:                                   {"no cache buffers available for reuse", 0, ""},
	336527575:                                   {"page @1, page type @2 lock conversion denied", 2, ""},
	336527576:                                   {"page @1, page type @2 lock denied", 2, ""},
	336527577:                                   {"buffer marked for update", 0, ""},
	336527578:                                   {"CCH: @1, status = @2 (218)", 2, ""},
	336527579:                                   {"request of unknown resource", 0, ""},
	336527580:                                   {"release of unknown resource", 0, ""},
	336527581:                                   {"(CMP) copy: cannot remap", 0, ""},
	336527582:                                   {"bad BLR -- invalid stream", 0, ""},
	336527583:                                   {"argument of scalar operation must be an array", 0, ""},
	336527584:                                   {"quad word arithmetic not supported", 0, ""},
	336527585:                                   {"data type not supported for arithmetic", 0, ""},
	336527586:                                   {"request size limit exceeded", 0, ""},
	336527587:                                   {"cannot access field @1 in view @2", 2, ""},
	336527588:                                   {"cannot access field in view @1", 1, ""},
	336527589:                                   {"EVL_assign_to: invalid operation", 0, ""},
	336527590:                                   {"EVL_bitmap: invalid operation", 0, ""},
	336527591:                                   {"EVL_boolean: invalid operation", 0, ""},
	336527592:                                   {"EVL_expr: invalid operation", 0, ""},
	336527593:                                   {"eval_statistical: invalid operation", 0, ""},
	336527594:                                   {"Unimplemented conversion, FAO directive O,Z,S", 0, ""},
	336527595:                                   {"Unimplemented conversion, FAO directive X,U", 0, ""},
	336527596:                                   {"Error parsing RDB FAO msg string", 0, ""},
	336527597:                                   {"Error parsing RDB FAO msg str", 0, ""},
	336527598:                                   {"unknown parameter in RdB status vector", 0, ""},
	336527599:                                   {"Red Database status vector inconsistent", 0, ""},
	336527600:                                   {"Red Database message parameter inconsistency", 0, ""},
	336527601:                                   {"error parsing RDB FAO message string", 0, ""},
	336527602:                                   {"unimplemented FAO directive", 0, ""},
	336527603:                                   {"missing pointer page in DPM_data_pages", 0, ""},
	336527604:                                   {"Fragment does not exist", 0, ""},
	336527605:                                   {"pointer page disappeared in DPM_delete", 0, ""},
	336527606:                                   {"pointer page lost from DPM_delete_relation", 0, ""},
	336527607:                                   {"missing pointer page in DPM_dump", 0, ""},
	336527608:                                   {"cannot find record fragment", 0, ""},
	336527609:                                   {"pointer page vanished from DPM_next", 0, ""},
	336527610:                                   {"temporary page buffer too small", 0, ""},
	336527611:                                   {"damaged data page", 0, ""},
	336527612:                                   {"header fragment length changed", 0, ""},
	336527613:                                   {"pointer page vanished from extend_relation", 0, ""},
	336527614:                                   {"pointer page vanished from relation list in locate_space", 0, ""},
	336527615:                                   {"cannot find free space", 0, ""},
	336527616:                                   {"pointer page vanished from mark_full", 0, ""},
	336527617:                                   {"bad record in RDB$PAGES", 0, ""},
	336527618:                                   {"page slot not empty", 0, ""},
	336527619:                                   {"bad pointer page", 0, ""},
	336527620:                                   {"index unexpectedly deleted", 0, ""},
	336527621:                                   {"scalar operator used on field which is not an array", 0, ""},
	336527622:                                   {"active", 0, ""},
	336527623:                                   {"committed", 0, ""},
	336527624:                                   {"rolled back", 0, ""},
	336527625:                                   {"in an ill-defined state", 0, ""},
	336527626:                                   {"next transaction older than oldest active transaction", 0, ""},
	336527627:                                   {"next transaction older than oldest transaction", 0, ""},
	336527628:                                   {"buffer marked during cache unwind", 0, ""},
	336527629:                                   {"error in recovery! database corrupted", 0, ""},
	336527630:                                   {"error in recovery! wrong data page record", 0, ""},
	336527631:                                   {"error in recovery! no space on data page", 0, ""},
	336527632:                                   {"error in recovery! wrong header page record", 0, ""},
	336527633:                                   {"error in recovery! wrong generator page record", 0, ""},
	336527634:                                   {"error in recovery! wrong b-tree page record", 0, ""},
	336527635:                                   {"error in recovery! wrong page inventory page record", 0, ""},
	336527636:                                   {"error in recovery! wrong pointer page record", 0, ""},
	336527637:                                   {"error in recovery! wrong index root page record", 0, ""},
	336527638:                                   {"error in recovery! wrong transaction page record", 0, ""},
	336527639:                                   {"error in recovery! out of sequence log record encountered", 0, ""},
	336527640:                                   {"error in recovery! unknown page type", 0, ""},
	336527641:                                   {"error in recovery! unknown record type", 0, ""},
	336527642:                                   {"journal server cannot archive to specified archive directory", 0, ""},
	336527643:                                   {"checksum error in log record when reading from log file", 0, ""},
	336527644:                                   {"cannot restore singleton select data", 0, ""},
	336527645:                                   {"lock not found in internal lock manager", 0, ""},
	336527646:                                   {"size of opt block exceeded", 0, ""},
	336527647:                                   {"Too many savepoints", 0, ""},
	336527648:                                   {"garbage collect record disappeared", 0, ""},
	336527649:                                   {"Unknown BLOB FILTER ACTION_", 0, ""},
	Isc_savepoint_error:                         {"error during savepoint backout", 0, ""},
	336527651:                                   {"cannot find record back version", 0, ""},
	336527652:                                   {"Illegal user_type.", 0, ""},
	336527653:                                   {"bad ACL", 0, ""},
	336527654:                                   {"inconsistent LATCH_mark release", 0, ""},
	336527655:                                   {"inconsistent LATCH_mark call", 0, ""},
	336527656:                                   {"inconsistent latch downgrade call", 0, ""},
	336527657:                                   {"bdb is unexpectedly marked", 0, ""},
	336527658:                                   {"missing exclusive latch", 0, ""},
	336527659:                                   {"exceeded maximum number of shared latches on a bdb", 0, ""},
	336527660:                                   {"can't find shared latch", 0, ""},
	Isc_cache_non_zero_use_count:                {"Non-zero use_count of a buffer in the empty que", 0, ""},
	Isc_unexpected_page_change:                  {"Unexpected page change from latching", 0, ""},
	336527663:                                   {"Invalid expression for evaluation", 0, ""},
	Isc_rdb_triggers_rdb_flags_corrupt:          {"RDB$FLAGS for trigger @1 in RDB$TRIGGERS is corrupted", 1, ""},
	336527665:                                   {"Blobs accounting is inconsistent", 0, ""},
	336527666:                                   {"Found array data type with more than 16 dimensions", 0, ""},
	336528361:                                   {"Cannot find system index, try to reconnect", 0, ""},
	Isc_GEN_ERR:                                 {"Statement failed, SQLSTATE = @1", 1, ""},
	Isc_USAGE:                                   {"usage:    isql [options] [<database>]", 0, ""},
	Isc_SWITCH:                                  {"Unknown switch: @1", 1, ""},
	Isc_NO_DB:                                   {"Use CONNECT or CREATE DATABASE to specify a database", 0, ""},
	Isc_FILE_OPEN_ERR:                           {"Unable to open @1", 1, ""},
	Isc_COMMIT_PROMPT:                           {"Commit current transaction (y/n)?", 0, ""},
	Isc_COMMIT_MSG:                              {"Committing.", 0, ""},
	Isc_ROLLBACK_MSG:                            {"Rolling back work.", 0, ""},
	Isc_CMD_ERR:                                 {"Command error: @1", 1, ""},
	Isc_ADD_PROMPT:                              {"Enter data or NULL for each column.  RETURN to end.", 0, ""},
	Isc_VERSION:                                 {"ISQL Version: @1", 1, ""},
	Isc_USAGE_ALL:                               {"\t-a(ll)                  extract metadata incl. legacy non-SQL tables", 0, ""},
	Isc_NUMBER_PAGES:                            {"Number of DB pages allocated = @1", 1, ""},
	Isc_SWEEP_INTERV:                            {"Sweep interval = @1", 1, ""},
	Isc_NUM_WAL_BUFF:                            {"Number of wal buffers = @1", 1, ""},
	Isc_WAL_BUFF_SIZE:                           {"Wal buffer size = @1", 1, ""},
	Isc_CKPT_LENGTH:                             {"Check point length = @1", 1, ""},
	Isc_CKPT_INTERV:                             {"Check point interval = @1", 1, ""},
	Isc_WAL_GRPC_WAIT:                           {"Wal group commit wait = @1", 1, ""},
	Isc_BASE_LEVEL:                              {"Base level = @1", 1, ""},
	Isc_LIMBO:                                   {"Transaction in limbo = @1", 1, ""},
	Isc_HLP_FRONTEND:                            {"Frontend commands:", 0, ""},
	Isc_HLP_BLOBED:                              {"BLOBVIEW <blobid>          -- view BLOB in text editor", 0, ""},
	Isc_HLP_BLOBDMP:                             {"BLOBDUMP <blobid> <file>   -- dump BLOB to a file", 0, ""},
	Isc_HLP_EDIT:                                {"EDIT     [<filename>]      -- edit SQL script file and execute", 0, ""},
	Isc_HLP_INPUT:                               {"INput    <filename>        -- take input from the named SQL file", 0, ""},
	Isc_HLP_OUTPUT:                              {"OUTput   [<filename>]      -- write output to named file", 0, ""},
	Isc_HLP_SHELL:                               {"SHELL    <command>         -- execute Operating System command in sub-shell", 0, ""},
	Isc_HLP_HELP:                                {"HELP                       -- display this menu", 0, ""},
	Isc_HLP_SETCOM:                              {"Set commands:", 0, ""},
	Isc_HLP_SET:                                 {"    SET                    -- display current SET options", 0, ""},
	Isc_HLP_SETAUTO:                             {"    SET AUTOddl            -- toggle autocommit of DDL statements", 0, ""},
	Isc_HLP_SETBLOB:                             {"    SET BLOB [ALL|<n>]     -- display BLOBS of subtype <n> or ALL", 0, ""},
	Isc_HLP_SETCOUNT:                            {"    SET COUNT              -- toggle count of selected rows on/off", 0, ""},
	Isc_HLP_SETECHO:                             {"    SET ECHO               -- toggle command echo on/off", 0, ""},
	Isc_HLP_SETSTAT:                             {"    SET STATs              -- toggle display of performance statistics", 0, ""},
	Isc_HLP_SETTERM:                             {"    SET TERM <string>      -- change statement terminator string", 0, ""},
	Isc_HLP_SHOW:                                {"SHOW     <object> [<name>|<pattern>] -- display system information", 0, ""},
	Isc_HLP_OBJTYPE:                             {"    <object> = ALL, CHECK, COLLATION, DATABASE, DOMAIN, EXCEPTION, FILTER, FUNCTION,", 0, ""},
	Isc_HLP_EXIT:                                {"EXIT                       -- exit and commit changes", 0, ""},
	Isc_HLP_QUIT:                                {"QUIT                       -- exit and roll back changes", 0, ""},
	Isc_HLP_ALL:                                 {"All commands may be abbreviated to letters in CAPitals", 0, ""},
	Isc_HLP_SETSCHEMA:                           {"\tSET SCHema/DB <db name> -- changes current database", 0, ""},
	Isc_YES_ANS:                                 {"Yes", 0, ""},
	Isc_REPORT1:                                 {"Current memory = !c\nDelta memory = !d\nMax memory = !x\nElapsed time = !e sec", 0, ""},
	Isc_REPORT2:                                 {"Cpu = !u sec\nBuffers = !b\nReads = !r\nWrites = !w\nFetches = !f", 0, ""},
	Isc_BLOB_SUBTYPE:                            {"BLOB display set to subtype @1. This BLOB: subtype = @2", 2, ""},
	Isc_BLOB_PROMPT:                             {"BLOB: @1, type 'edit' or filename to load>", 1, ""},
	Isc_DATE_PROMPT:                             {"Enter @1 as Y/M/D>", 1, ""},
	Isc_NAME_PROMPT:                             {"Enter @1>", 1, ""},
	Isc_DATE_ERR:                                {"Bad date @1", 1, ""},
	Isc_CON_PROMPT:                              {"CON> ", 0, ""},
	Isc_HLP_SETLIST:                             {"    SET LIST               -- toggle column or table display format", 0, ""},
	Isc_NOT_FOUND:                               {"@1 not found", 1, ""},
	Isc_COPY_ERR:                                {"Errors occurred (possibly duplicate domains) in creating @1 in @2", 2, ""},
	Isc_SERVER_TOO_OLD:                          {"Server version too old to support the isql command", 0, ""},
	Isc_REC_COUNT:                               {"Records affected: @1", 1, ""},
	Isc_UNLICENSED:                              {"Unlicensed for database \"@1\"", 1, ""},
	Isc_HLP_SETWIDTH:                            {"    SET WIDTH <col> [<n>]  -- set/unset print width to <n> for column <col>", 0, ""},
	Isc_HLP_SETPLAN:                             {"    SET PLAN               -- toggle display of query access plan", 0, ""},
	Isc_HLP_SETTIME:                             {"    SET TIME               -- toggle display of timestamp with DATE values", 0, ""},
	Isc_HLP_EDIT2:                               {"EDIT                       -- edit current command buffer and execute", 0, ""},
	Isc_HLP_OUTPUT2:                             {"OUTput                     -- return output to stdout", 0, ""},
	Isc_HLP_SETNAMES:                            {"    SET NAMES <csname>     -- set name of runtime character set", 0, ""},
	Isc_HLP_OBJTYPE2:                            {"               GENERATOR, GRANT, INDEX, PACKAGE, PROCEDURE, ROLE, SQL DIALECT,", 0, ""},
	Isc_HLP_SETBLOB2:                            {"    SET BLOB               -- turn off BLOB display", 0, ""},
	Isc_HLP_SET_ROOT:                            {"SET      <option>          -- (Use HELP SET for complete list)", 0, ""},
	Isc_NO_TABLES:                               {"There are no tables in this database", 0, ""},
	Isc_NO_TABLE:                                {"There is no table @1 in this database", 1, ""},
	Isc_NO_VIEWS:                                {"There are no views in this database", 0, ""},
	Isc_NO_VIEW:                                 {"There is no view @1 in this database", 1, ""},
	Isc_NO_INDICES_ON_REL:                       {"There are no indices on table @1 in this database", 1, ""},
	Isc_NO_REL_OR_INDEX:                         {"There is no table or index @1 in this database", 1, ""},
	Isc_NO_INDICES:                              {"There are no indices in this database", 0, ""},
	Isc_NO_DOMAIN:                               {"There is no domain @1 in this database", 1, ""},
	Isc_NO_DOMAINS:                              {"There are no domains in this database", 0, ""},
	Isc_NO_EXCEPTION:                            {"There is no exception @1 in this database", 1, ""},
	Isc_NO_EXCEPTIONS:                           {"There are no exceptions in this database", 0, ""},
	Isc_NO_FILTER:                               {"There is no filter @1 in this database", 1, ""},
	Isc_NO_FILTERS:                              {"There are no filters in this database", 0, ""},
	Isc_NO_FUNCTION:                             {"There is no user-defined function @1 in this database", 1, ""},
	Isc_NO_FUNCTIONS:                            {"There are no user-defined functions in this database", 0, ""},
	Isc_NO_GEN:                                  {"There is no generator @1 in this database", 1, ""},
	Isc_NO_GENS:                                 {"There are no generators in this database", 0, ""},
	Isc_NO_GRANT_ON_REL:                         {"There is no privilege granted on table @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_PROC:                        {"There is no privilege granted on stored procedure @1 in this database", 1, ""},
	Isc_NO_REL_OR_PROC:                          {"There is no table or stored procedure @1 in this database", 1, ""},
	Isc_NO_PROC:                                 {"There is no stored procedure @1 in this database", 1, ""},
	Isc_NO_PROCS:                                {"There are no stored procedures in this database", 0, ""},
	Isc_NO_TRIGGERS_ON_REL:                      {"There are no triggers on table @1 in this database", 1, ""},
	Isc_NO_REL_OR_TRIGGER:                       {"There is no table or trigger @1 in this database", 1, ""},
	Isc_NO_TRIGGERS:                             {"There are no triggers in this database", 0, ""},
	Isc_NO_CHECKS_ON_REL:                        {"There are no check constraints on table @1 in this database", 1, ""},
	Isc_REPORT2_WINDOWS_ONLY:                    {"Buffers = !b\nReads = !r\nWrites !w\nFetches = !f", 0, ""},
	Isc_BUFFER_OVERFLOW:                         {"Single isql command exceeded maximum buffer size", 0, ""},
	Isc_NO_ROLES:                                {"There are no roles in this database", 0, ""},
	Isc_NO_OBJECT:                               {"There is no metadata object @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_ROL:                         {"There is no membership privilege granted on @1 in this database", 1, ""},
	Isc_UNEXPECTED_EOF:                          {"Expected end of statement, encountered EOF", 0, ""},
	Isc_TIME_ERR:                                {"Bad TIME: @1", 1, ""},
	Isc_HLP_OBJTYPE3:                            {"               SYSTEM, TABLE, TRIGGER, VERSION, USERS, VIEW, WIRE_STATISTICS", 0, ""},
	Isc_NO_ROLE:                                 {"There is no role @1 in this database", 1, ""},
	Isc_USAGE_BAIL:                              {"\t-b(ail)                 bail on errors (set bail on)", 0, ""},
	336658537:                                   {"Incomplete string in @1", 1, ""},
	Isc_HLP_SETSQLDIALECT:                       {"    SET SQL DIALECT <n>    -- set sql dialect to <n>", 0, ""},
	Isc_NO_GRANT_ON_ANY:                         {"There is no privilege granted in this database", 0, ""},
	Isc_HLP_SETPLANONLY:                         {"    SET PLANONLY           -- toggle display of query plan without executing", 0, ""},
	Isc_HLP_SETHEADING:                          {"    SET HEADING            -- toggle display of query column titles", 0, ""},
	Isc_HLP_SETBAIL:                             {"    SET BAIL               -- toggle bailing out on errors in non-interactive mode", 0, ""},
	Isc_USAGE_CACHE:                             {"\t-c(ache) <num>          number of cache buffers", 0, ""},
	Isc_TIME_PROMPT:                             {"Enter @1 as H:M:S>", 1, ""},
	Isc_TIMESTAMP_PROMPT:                        {"Enter @1 as Y/MON/D H:MIN:S[.MSEC]>", 1, ""},
	Isc_TIMESTAMP_ERR:                           {"Bad TIMESTAMP: @1", 1, ""},
	Isc_NO_COMMENTS:                             {"There are no comments for objects in this database", 0, ""},
	Isc_ONLY_FIRST_BLOBS:                        {"Printing only the first @1 blobs.", 1, ""},
	Isc_MSG_TABLES:                              {"Tables:", 0, ""},
	Isc_MSG_FUNCTIONS:                           {"Functions:", 0, ""},
	Isc_EXACTLINE:                               {"At line @1 in file @2", 2, ""},
	Isc_AFTERLINE:                               {"After line @1 in file @2", 2, ""},
	Isc_NO_TRIGGER:                              {"There is no trigger @1 in this database", 1, ""},
	Isc_USAGE_CHARSET:                           {"\t-ch(arset) <charset>    connection charset (set names)", 0, ""},
	Isc_USAGE_DATABASE:                          {"\t-d(atabase) <database>  database name to put in script creation", 0, ""},
	Isc_USAGE_ECHO:                              {"\t-e(cho)                 echo commands (set echo on)", 0, ""},
	Isc_USAGE_EXTRACT:                           {"\t-ex(tract)              extract metadata", 0, ""},
	Isc_USAGE_INPUT:                             {"\t-i(nput) <file>         input file (set input)", 0, ""},
	Isc_USAGE_MERGE:                             {"\t-m(erge)                merge standard error", 0, ""},
	Isc_USAGE_MERGE2:                            {"\t-m2                     merge diagnostic", 0, ""},
	Isc_USAGE_NOAUTOCOMMIT:                      {"\t-n(oautocommit)         no autocommit DDL (set autoddl off)", 0, ""},
	Isc_USAGE_NOWARN:                            {"\t-now(arnings)           do not show warnings", 0, ""},
	Isc_USAGE_OUTPUT:                            {"\t-o(utput) <file>        output file (set output)", 0, ""},
	Isc_USAGE_PAGE:                              {"\t-pag(elength) <size>    page length", 0, ""},
	Isc_USAGE_PASSWORD:                          {"\t-p(assword) <password>  connection password", 0, ""},
	Isc_USAGE_QUIET:                             {"\t-q(uiet)                do not show the message \"Use CONNECT...\"", 0, ""},
	Isc_USAGE_ROLE:                              {"\t-r(ole) <role>          role name", 0, ""},
	Isc_USAGE_ROLE2:                             {"\t-r2 <role>              role (uses quoted identifier)", 0, ""},
	Isc_USAGE_SQLDIALECT:                        {"\t-s(qldialect) <dialect> SQL dialect (set sql dialect)", 0, ""},
	Isc_USAGE_TERM:                              {"\t-t(erminator) <term>    command terminator (set term)", 0, ""},
	Isc_USAGE_USER:                              {"\t-u(ser) <user>          user name", 0, ""},
	Isc_USAGE_XTRACT:                            {"\t-x                      extract metadata", 0, ""},
	Isc_USAGE_VERSION:                           {"\t-z                      show program and server version", 0, ""},
	Isc_USAGE_NOARG:                             {"missing argument for switch \"@1\"", 1, ""},
	Isc_USAGE_NOTINT:                            {"argument \"@1\" for switch \"@2\" is not an integer", 2, ""},
	Isc_USAGE_RANGE:                             {"value \"@1\" for switch \"@2\" is out of range", 2, ""},
	Isc_USAGE_DUPSW:                             {"switch \"@1\" or its equivalent used more than once", 1, ""},
	Isc_USAGE_DUPDB:                             {"more than one database name: \"@1\", \"@2\"", 2, ""},
	Isc_NO_DEPENDENCIES:                         {"No dependencies for @1 were found", 1, ""},
	Isc_NO_COLLATION:                            {"There is no collation @1 in this database", 1, ""},
	Isc_NO_COLLATIONS:                           {"There are no user-defined collations in this database", 0, ""},
	Isc_MSG_COLLATIONS:                          {"Collations:", 0, ""},
	Isc_NO_SECCLASS:                             {"There are no security classes for @1", 1, ""},
	Isc_NO_DB_WIDE_SECCLASS:                     {"There is no database-wide security class", 0, ""},
	Isc_CANNOT_GET_SRV_VER:                      {"Cannot get server version without database connection", 0, ""},
	Isc_USAGE_NODBTRIGGERS:                      {"\t-nod(btriggers)         do not run database triggers", 0, ""},
	Isc_USAGE_TRUSTED:                           {"\t-tr(usted)              use trusted authentication", 0, ""},
	Isc_BULK_PROMPT:                             {"BULK> ", 0, ""},
	Isc_NO_CONNECTED_USERS:                      {"There are no connected users", 0, ""},
	Isc_USERS_IN_DB:                             {"Users in the database", 0, ""},
	Isc_OUTPUT_TRUNCATED:                        {"Output was truncated", 0, ""},
	Isc_VALID_OPTIONS:                           {"Valid options are:", 0, ""},
	Isc_USAGE_FETCH:                             {"\t-f(etch_password)       fetch password from file", 0, ""},
	Isc_PASS_FILE_OPEN:                          {"could not open password file @1, errno @2", 2, ""},
	Isc_PASS_FILE_READ:                          {"could not read password file @1, errno @2", 2, ""},
	Isc_EMPTY_PASS:                              {"empty password file @1", 1, ""},
	Isc_HLP_SETMAXROWS:                          {"    SET MAXROWS [<n>]      -- limit select stmt to <n> rows, zero is no limit", 0, ""},
	Isc_NO_PACKAGE:                              {"There is no package @1 in this database", 1, ""},
	Isc_NO_PACKAGES:                             {"There are no packages in this database", 0, ""},
	Isc_NO_SCHEMA:                               {"There is no schema @1 in this database", 1, ""},
	Isc_NO_SCHEMAS:                              {"There are no schemas in this database", 0, ""},
	Isc_MAXROWS_INVALID:                         {"Unable to convert @1 to a number for MAXROWS option", 1, ""},
	Isc_MAXROWS_OUTOF_RANGE:                     {"Value @1 for MAXROWS is out of range. Max value is @2", 2, ""},
	Isc_MAXROWS_NEGATIVE:                        {"The value (@1) for MAXROWS must be zero or greater", 1, ""},
	Isc_HLP_SETEXPLAIN:                          {"    SET EXPLAIN            -- toggle display of query access plan in the explained form", 0, ""},
	Isc_NO_GRANT_ON_GEN:                         {"There is no privilege granted on generator @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_XCP:                         {"There is no privilege granted on exception @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_FLD:                         {"There is no privilege granted on domain @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_CS:                          {"There is no privilege granted on character set @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_COLL:                        {"There is no privilege granted on collation @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_PKG:                         {"There is no privilege granted on package @1 in this database", 1, ""},
	Isc_NO_GRANT_ON_FUN:                         {"There is no privilege granted on function @1 in this database", 1, ""},
	Isc_REPORT_NEW1:                             {"Current memory = !\nDelta memory = !\nMax memory = !\nElapsed time = ~ sec", 0, ""},
	Isc_REPORT_NEW2:                             {"Cpu = ~ sec", 0, ""},
	Isc_REPORT_NEW3:                             {"Buffers = !\nReads = !\nWrites = !\nFetches = !", 0, ""},
	Isc_NO_MAP:                                  {"There is no mapping @1 in this database", 1, ""},
	Isc_NO_MAPS:                                 {"There are no mappings in this database", 0, ""},
	Isc_INVALID_TERM_CHARS:                      {"Invalid characters for SET TERMINATOR are @1", 1, ""},
	Isc_REC_DISPLAYCOUNT:                        {"Records displayed: @1", 1, ""},
	Isc_COLUMNS_HIDDEN:                          {"Full NULL columns hidden due to RecordBuff: @1", 1, ""},
	Isc_HLP_SETRECORDBUF:                        {"    SET RECORDBuf          -- toggle limited buffering and trimming of columns", 0, ""},
	Isc_NUMBER_USED_PAGES:                       {"Number of DB pages used = @1", 1, ""},
	Isc_NUMBER_FREE_PAGES:                       {"Number of DB pages free = @1", 1, ""},
	Isc_DATABASE_CRYPTED:                        {"Database encrypted", 0, ""},
	Isc_DATABASE_NOT_CRYPTED:                    {"Database not encrypted", 0, ""},
	Isc_DATABASE_CRYPT_PROCESS:                  {"crypt thread not complete", 0, ""},
	Isc_MSG_ROLES:                               {"Roles:", 0, ""},
	Isc_NO_TIMEOUTS:                             {"Timeouts are not supported by server", 0, ""},
	Isc_HLP_SETKEEPTRAN:                         {"    SET KEEP_TRAN_params   -- toggle to keep or not to keep text of following successful SET TRANSACTION statement", 0, ""},
	Isc_HLP_SETPERTAB:                           {"    SET PER_TABle_stats    -- toggle display of detailed per-table statistics", 0, ""},
	Isc_BAD_STMT_TYPE:                           {"Statement type is not recognized", 0, ""},
	Isc_MSG_PACKAGES:                            {"Packages:", 0, ""},
	Isc_NO_PUBLICATION:                          {"There is no publication @1 in this database", 1, ""},
	Isc_NO_PUBLICATIONS:                         {"There is no publications in this database", 0, ""},
	Isc_MSG_PUBLICATIONS:                        {"Publications:", 0, ""},
	Isc_MSG_PROCEDURES:                          {"Procedures:", 0, ""},
	Isc_HLP_EXPLAIN:                             {"EXPLAIN                    -- explain a query access plan", 0, ""},
	Isc_USAGE_AUTOTERM:                          {"\t-autot(erm)             use auto statement terminator (set autoterm on)", 0, ""},
	Isc_AUTOTERM_NOT_SUPPORTED:                  {"SET AUTOTERM ON is not supported in engine/server and has been disabled", 0, ""},
	Isc_HLP_SETAUTOTERM:                         {"    SET AUTOTERM           -- toggle auto statement terminator", 0, ""},
	Isc_HLP_SETWIRESTATS:                        {"    SET WIRE_stats         -- toggle display of wire (network) statistics", 0, ""},
	Isc_USAGE_SEARCH_PATH:                       {"\t-(se)arch_path <path>   set schema search path", 0, ""},
	Isc_MSG_SCHEMAS:                             {"Schemas:", 0, ""},
	Isc_NO_CONSTANTS:                            {"There are no constants in this database", 0, ""},
	Isc_NO_CONSTANT:                             {"There are no constant @1 in this database", 1, ""},
	Isc_NO_OBJECT_BY_PATTERN:                    {"Pattern @1 does not match any object", 1, ""},
	Isc_MSG_GENERATORS:                          {"Generators:", 0, ""},
	Isc_MSG_VIEWS:                               {"Views:", 0, ""},
	Isc_MSG_INDICIES:                            {"Indicies:", 0, ""},
	Isc_MSG_CONSTRAINS:                          {"Constrains:", 0, ""},
	Isc_MSG_DOMAINS:                             {"Domains:", 0, ""},
	Isc_MSG_TRIGGERS:                            {"Triggers:", 0, ""},
	Isc_MSG_EXCEPTIONS:                          {"Exceptions:", 0, ""},
	Isc_MSG_GRANTS:                              {"Grants:", 0, ""},
	Isc_MSG_COMMENTS:                            {"Comments:", 0, ""},
	Isc_MSG_FILTERS:                             {"Filters:", 0, ""},
	Isc_MSG_DEPENDENCIES:                        {"Dependencies:", 0, ""},
	Isc_MSG_SECCLASSES:                          {"Security classes:", 0, ""},
	Isc_MSG_SYSTEM:                              {"System:", 0, ""},
	Isc_MSG_MAP:                                 {"Maps:", 0, ""},
	Isc_USAGE_CERTIFICATE:                       {"\t-ce(rtificate) <alias>  use certificate for authentication", 0, ""},
	Isc_USAGE_PIN:                               {"\t-pi(n) <PIN>            PIN for certificate's private key", 0, ""},
	Isc_USAGE_VERIFY:                            {"\t-v(erify) <alias>       verify server certificate", 0, ""},
	Isc_USAGE_LOGIN:                             {"\t-l(ogin)                effective login for trusted user", 0, ""},
	Isc_CLEAR_HISTORY:                           {"\t-cl(earhistory)         clear command history", 0, ""},
	Isc_MSG_SYS_FUNCTIONS:                       {"System Functions:", 0, ""},
	Isc_MSG_SYS_PROCEDURES:                      {"System Procedures:", 0, ""},
	Isc_MSG_SYS_GENERATORS:                      {"System Generators:", 0, ""},
	Isc_MSG_SYS_COLLATIONS:                      {"System Collations:", 0, ""},
	Isc_MSG_SYS_PACKAGES:                        {"System Packages:", 0, ""},
	Isc_MSG_SYS_PUBLICATIONS:                    {"System Publications:", 0, ""},
	Isc_MSG_SYS_TABLES:                          {"System Tables:", 0, ""},
	Isc_MSG_JOBS:                                {"Jobs:", 0, ""},
	Isc_NO_JOB:                                  {"There is no job @1", 1, ""},
	Isc_NO_JOBS:                                 {"There are no jobs", 0, ""},
	Isc_NO_SYSTEM_COLLATIONS:                    {"There are no system collations in this database", 0, ""},
	Isc_NO_SYSTEM_FUNCTIONS:                     {"There are no system functions in this database", 0, ""},
	Isc_NO_SYSTEM_TABLES:                        {"There are no system tables in this database", 0, ""},
	Isc_NO_SYSTEM_GENS:                          {"There are no system generators in this database", 0, ""},
	Isc_NO_SYSTEM_PROCS:                         {"There are no system procedures in this database", 0, ""},
	Isc_NO_SYSTEM_PACKAGES:                      {"There are no system packages in this database", 0, ""},
	Isc_NO_SYSTEM_PUBLICATIONS:                  {"There is no system publications in this database", 0, ""},
	Isc_NO_SYSTEM_COLLATION:                     {"There is no system collation @1 in this database", 1, ""},
	Isc_NO_SYSTEM_FUNCTION:                      {"There is no system function @1 in this database", 1, ""},
	Isc_NO_SYSTEM_TABLE:                         {"There is no system table @1 in this database", 1, ""},
	Isc_NO_SYSTEM_GEN:                           {"There is no system generator @1 in this database", 1, ""},
	Isc_NO_SYSTEM_PROC:                          {"There is no system procedure @1 in this database", 1, ""},
	Isc_NO_SYSTEM_PACKAGE:                       {"There in no system package @1 in this database", 1, ""},
	Isc_NO_SYSTEM_PUBLICATION:                   {"There is no system publication @1 in this database", 1, ""},
	Isc_NO_ANY_DEPENDENCIES:                     {"No dependencies were found", 0, ""},
	Isc_MSG_SYS_ROLES:                           {"System Roles:", 0, ""},
	Isc_NO_SYSTEM_ROLE:                          {"There is no system role @1 in this database", 1, ""},
	Isc_USAGE_EXTRACT_NO_TERM:                   {"\t-xn                     extract metadata without usage \"SET TERM\"", 0, ""},
	Isc_USAGE_TOKEN:                             {"\t-to(ken) <token>        access token for OpenID Connect", 0, ""},
	Isc_MSG_FOREIGN_SERVERS:                     {"Servers:", 0, ""},
	Isc_MSG_FOREIGN_USER_MAPPINGS:               {"Foreign User Mappings:", 0, ""},
	Isc_NO_FOREIGN_SERVER:                       {"There is no server @1 in this database", 1, ""},
	Isc_NO_FOREIGN_SERVERS:                      {"There are no servers in this database", 0, ""},
	Isc_NO_FOREIGN_USER_MAPPING:                 {"There is no user mappings for @1 in this database", 1, ""},
	Isc_NO_FOREIGN_USER_MAPPINGS:                {"There is no user mappings in this database", 0, ""},
	Isc_USAGE_READ:                              {"\t-re(ad_password)        request password from user", 0, ""},
	Isc_GsecMsg1:                                {"GSEC>", 0, ""},
	Isc_GsecMsg2:                                {"gsec", 0, ""},
	Isc_GsecMsg3:                                {"ADD            add user", 0, ""},
	Isc_GsecMsg4:                                {"DELETE         delete user", 0, ""},
	Isc_GsecMsg5:                                {"DISPLAY        display user(s)", 0, ""},
	Isc_GsecMsg6:                                {"MODIFY         modify user", 0, ""},
	Isc_GsecMsg7:                                {"PW             user's password", 0, ""},
	Isc_GsecMsg8:                                {"UID            user's ID", 0, ""},
	Isc_GsecMsg9:                                {"GID            user's group ID", 0, ""},
	Isc_GsecMsg10:                               {"PROJ           user's project name", 0, ""},
	Isc_GsecMsg11:                               {"ORG            user's organization name", 0, ""},
	Isc_GsecMsg12:                               {"FNAME          user's first name", 0, ""},
	Isc_GsecMsg13:                               {"MNAME          user's middle name/initial", 0, ""},
	Isc_GsecMsg14:                               {"LNAME          user's last name", 0, ""},
	Isc_gsec_cant_open_db:                       {"unable to open database", 0, "00000"},
	Isc_gsec_switches_error:                     {"error in switch specifications", 0, "00000"},
	Isc_gsec_no_op_spec:                         {"no operation specified", 0, "00000"},
	Isc_gsec_no_usr_name:                        {"no user name specified", 0, "00000"},
	Isc_gsec_err_add:                            {"add record error", 0, "00000"},
	Isc_gsec_err_modify:                         {"modify record error", 0, "00000"},
	Isc_gsec_err_find_mod:                       {"find/modify record error", 0, "00000"},
	Isc_gsec_err_rec_not_found:                  {"record not found for user: @1", 1, "00000"},
	Isc_gsec_err_delete:                         {"delete record error", 0, "00000"},
	Isc_gsec_err_find_del:                       {"find/delete record error", 0, "00000"},
	Isc_GsecMsg25:                               {"users defined for node", 0, ""},
	Isc_GsecMsg26:                               {"     user name                    uid   gid admin     full name", 0, ""},
	Isc_GsecMsg27:                               {"------------------------------------------------------------------------------------------------", 0, ""},
	Isc_gsec_err_find_disp:                      {"find/display record error", 0, "00000"},
	Isc_gsec_inv_param:                          {"invalid parameter, no switch defined", 0, "00000"},
	Isc_gsec_op_specified:                       {"operation already specified", 0, "00000"},
	Isc_gsec_pw_specified:                       {"password already specified", 0, "00000"},
	Isc_gsec_uid_specified:                      {"uid already specified", 0, "00000"},
	Isc_gsec_gid_specified:                      {"gid already specified", 0, "00000"},
	Isc_gsec_proj_specified:                     {"project already specified", 0, "00000"},
	Isc_gsec_org_specified:                      {"organization already specified", 0, "00000"},
	Isc_gsec_fname_specified:                    {"first name already specified", 0, "00000"},
	Isc_gsec_mname_specified:                    {"middle name already specified", 0, "00000"},
	Isc_gsec_lname_specified:                    {"last name already specified", 0, "00000"},
	Isc_GsecMsg39:                               {"gsec version", 0, ""},
	Isc_gsec_inv_switch:                         {"invalid switch specified", 0, "00000"},
	Isc_gsec_amb_switch:                         {"ambiguous switch specified", 0, "00000"},
	Isc_gsec_no_op_specified:                    {"no operation specified for parameters", 0, "00000"},
	Isc_gsec_params_not_allowed:                 {"no parameters allowed for this operation", 0, "00000"},
	Isc_gsec_incompat_switch:                    {"incompatible switches specified", 0, "00000"},
	Isc_GsecMsg45:                               {"gsec utility - maintains user password database", 0, ""},
	Isc_GsecMsg46:                               {"command line usage:", 0, ""},
	Isc_GsecMsg47:                               {"<command> [ <parameter> ... ]", 0, ""},
	Isc_GsecMsg48:                               {"interactive usage:", 0, ""},
	Isc_GsecMsg49:                               {"available commands:", 0, ""},
	Isc_GsecMsgs50:                              {"adding a new user:", 0, ""},
	Isc_GsecMsg51:                               {"add <name> [ <parameter> ... ]", 0, ""},
	Isc_GsecMsg52:                               {"deleting a current user:", 0, ""},
	Isc_GsecMsg53:                               {"delete <name>", 0, ""},
	Isc_GsecMsg54:                               {"displaying all users:", 0, ""},
	Isc_GsecMsg55:                               {"display", 0, ""},
	Isc_GsecMsg56:                               {"displaying one user:", 0, ""},
	Isc_GsecMsg57:                               {"display <name>", 0, ""},
	Isc_GsecMsg58:                               {"modifying a user's parameters:", 0, ""},
	Isc_GsecMsg59:                               {"modify <name> <parameter> [ <parameter> ... ]", 0, ""},
	Isc_GsecMsg60:                               {"help:", 0, ""},
	Isc_GsecMsg61:                               {"? (interactive only)", 0, ""},
	Isc_GsecMsg62:                               {"help", 0, ""},
	Isc_GsecMsg63:                               {"quit interactive session:", 0, ""},
	Isc_GsecMsg64:                               {"quit (interactive only)", 0, ""},
	Isc_GsecMsg65:                               {"available parameters:", 0, ""},
	Isc_GsecMsg66:                               {"-pw <password>", 0, ""},
	Isc_GsecMsg67:                               {"-uid <uid>", 0, ""},
	Isc_GsecMsg68:                               {"-gid <uid>", 0, ""},
	Isc_GsecMsg69:                               {"-proj <projectname>", 0, ""},
	Isc_GsecMsg70:                               {"-org <organizationname>", 0, ""},
	Isc_GsecMsg71:                               {"-fname <firstname>", 0, ""},
	Isc_GsecMsg72:                               {"-mname <middlename>", 0, ""},
	Isc_GsecMsg73:                               {"-lname <lastname>", 0, ""},
	336724042:                                   {"gsec - memory allocation error", 0, ""},
	336724043:                                   {"gsec error", 0, ""},
	Isc_gsec_inv_username:                       {"Invalid user name (maximum 31 bytes allowed)", 0, "00000"},
	Isc_gsec_inv_pw_length:                      {"Warning - maximum 8 significant bytes of password used", 0, "00000"},
	Isc_gsec_db_specified:                       {"database already specified", 0, "00000"},
	Isc_gsec_db_admin_specified:                 {"database administrator name already specified", 0, "00000"},
	Isc_gsec_db_admin_pw_specified:              {"database administrator password already specified", 0, "00000"},
	Isc_gsec_sql_role_specified:                 {"SQL role name already specified", 0, "00000"},
	Isc_GsecMsg82:                               {"[ <options> ... ]", 0, ""},
	Isc_GsecMsg83:                               {"available options:", 0, ""},
	Isc_GsecMsg84:                               {"-user <database administrator name>", 0, ""},
	Isc_GsecMsg85:                               {"-password <database administrator password>", 0, ""},
	Isc_GsecMsg86:                               {"-role <database administrator SQL role name>", 0, ""},
	Isc_GsecMsg87:                               {"-database <database to manage>", 0, ""},
	Isc_GsecMsg88:                               {"-z", 0, ""},
	Isc_GsecMsg89:                               {"displaying version number:", 0, ""},
	Isc_GsecMsg90:                               {"z (interactive only)", 0, ""},
	Isc_GsecMsg91:                               {"-trusted (use trusted authentication)", 0, ""},
	Isc_GsecMsg92:                               {"invalid switch specified in interactive mode", 0, ""},
	Isc_GsecMsg93:                               {"error closing security database", 0, ""},
	Isc_GsecMsg94:                               {"error releasing request in security database", 0, ""},
	Isc_GsecMsg95:                               {"-fetch_password <file to fetch password from>", 0, ""},
	Isc_GsecMsg96:                               {"error fetching password from file", 0, ""},
	Isc_GsecMsg97:                               {"error changing AUTO ADMINS MAPPING in security database", 0, ""},
	Isc_GsecMsg98:                               {"changing admins mapping to RDB$ADMIN role in security database:", 0, ""},
	Isc_GsecMsg99:                               {"invalid parameter for -MAPPING, only SET or DROP is accepted", 0, ""},
	Isc_GsecMsg100:                              {"mapping {set|drop}", 0, ""},
	Isc_GsecMsg101:                              {"use gsec -? to get help", 0, ""},
	Isc_GsecMsg102:                              {"-admin {yes|no}", 0, ""},
	Isc_GsecMsg103:                              {"invalid parameter for -ADMIN, only YES or NO is accepted", 0, ""},
	Isc_GsecMsg104:                              {"not enough privileges to complete operation", 0, ""},
	Isc_GsecMsg200:                              {"user not found in LDAP", 0, ""},
	Isc_GsecMsg201:                              {"error changing LDAP password", 0, ""},
	Isc_GsecMsg202:                              {"cannot bind to LDAP server", 0, ""},
	Isc_gstat_unknown_switch:                    {"found unknown switch", 0, "00000"},
	Isc_gstat_retry:                             {"please retry, giving a database name", 0, "00000"},
	Isc_gstat_wrong_ods:                         {"Wrong ODS version, expected @1, encountered @2", 2, "00000"},
	Isc_gstat_unexpected_eof:                    {"Unexpected end of database file.", 0, "00000"},
	336920581:                                   {"gstat version @1", 1, ""},
	336920582:                                   {"\nDatabase \"@1\"", 1, ""},
	336920583:                                   {"\n\nDatabase file sequence:", 0, ""},
	336920584:                                   {"File @1 continues as file @2", 2, ""},
	336920585:                                   {"File @1 is the @2 file", 2, ""},
	336920586:                                   {"\nAnalyzing database pages ...", 0, ""},
	336920587:                                   {"    Primary pointer page: @1, Index root page: @2", 2, ""},
	336920588:                                   {"    Data pages: @1, data page slots: @2, average fill: @3", 3, ""},
	336920589:                                   {"    Fill distribution:", 0, ""},
	336920590:                                   {"    Index @1 (@2)", 2, ""},
	336920591:                                   {"\tDepth: @1, leaf buckets: @2, nodes: @3", 3, ""},
	336920592:                                   {"\tAverage data length: @1, total dup: @2, max dup: @3", 3, ""},
	336920593:                                   {"\tFill distribution:", 0, ""},
	336920594:                                   {"    Expected data on page @1", 1, ""},
	336920595:                                   {"    Expected b-tree bucket on page @1 from @2", 2, ""},
	336920596:                                   {"unknown switch \"@1\"", 1, ""},
	336920597:                                   {"Available switches:", 0, ""},
	336920598:                                   {"    -a      analyze data and index pages", 0, ""},
	336920599:                                   {"    -d      analyze data pages", 0, ""},
	336920600:                                   {"    -h      analyze header page ONLY", 0, ""},
	336920601:                                   {"    -i      analyze index leaf pages", 0, ""},
	336920602:                                   {"    -l      analyze log page", 0, ""},
	336920603:                                   {"    -s      analyze system relations in addition to user tables", 0, ""},
	336920604:                                   {"    -z      display version number", 0, ""},
	Isc_gstat_open_err:                          {"Can't open database file @1", 1, "00000"},
	Isc_gstat_read_err:                          {"Can't read a database page", 0, "00000"},
	Isc_gstat_sysmemex:                          {"System memory exhausted", 0, "00000"},
	Isc_gstat_username:                          {"    -u      username", 0, ""},
	Isc_gstat_password:                          {"    -p      password", 0, ""},
	336920610:                                   {"    -r      analyze average record and version length", 0, ""},
	336920611:                                   {"    -t      tablename <tablename2...> (case sensitive)", 0, ""},
	336920612:                                   {"    -tr     use trusted authentication", 0, ""},
	336920613:                                   {"    -fetch  fetch password from file", 0, ""},
	336920614:                                   {"option -h is incompatible with options -a, -d, -i, -r, -schema, -s, -b and -t", 0, ""},
	336920615:                                   {"usage:   gstat [options] <database> or gstat <database> [options]", 0, ""},
	336920616:                                   {"database name was already specified", 0, ""},
	336920617:                                   {"option -t needs a table name", 0, ""},
	336920618:                                   {"option -t got a too long table name @1", 1, ""},
	336920619:                                   {"option -t accepts several table names only if used after <database>", 0, ""},
	336920620:                                   {"table \"@1\" not found", 1, ""},
	336920621:                                   {"use gstat -? to get help", 0, ""},
	336920622:                                   {"    Primary pages: @1, secondary pages: @2, swept pages: @3", 3, ""},
	336920623:                                   {"    Big record pages: @1", 1, ""},
	336920624:                                   {"    Blobs: @1, total length: @2, blob pages: @3", 3, ""},
	336920625:                                   {"        Level 0: @1, Level 1: @2, Level 2: @3", 3, ""},
	336920626:                                   {"option -e is incompatible with options -a, -d, -h, -i, -r, -schema, -s, -b and -t", 0, ""},
	336920627:                                   {"    -e      analyze database encryption", 0, ""},
	336920628:                                   {"Data pages: total @1, encrypted @2, non-crypted @3", 3, ""},
	336920629:                                   {"Index pages: total @1, encrypted @2, non-crypted @3", 3, ""},
	336920630:                                   {"Blob pages: total @1, encrypted @2, non-crypted @3", 3, ""},
	336920631:                                   {"no encrypted database support, only -e and -h can be used", 0, ""},
	336920632:                                   {"    Empty pages: @1, full pages: @2", 2, ""},
	336920633:                                   {"    -role   SQL role name", 0, ""},
	336920634:                                   {"Other pages: total @1, ENCRYPTED @2 (DB problem!), non-crypted @3", 3, ""},
	336920635:                                   {"Gstat execution time @1", 1, ""},
	336920636:                                   {"Gstat completion time @1", 1, ""},
	336920637:                                   {"    Expected page inventory page @1", 1, ""},
	336920638:                                   {"Generator pages: total @1, encrypted @2, non-crypted @3", 3, ""},
	336920639:                                   {"    Table size: @1 bytes", 1, ""},
	336920640:                                   {"        Level @1: @2, total length: @3, blob pages: @4", 4, ""},
	336920641:                                   {"    -sch    schemaname <schemaname2...> (case sensitive)", 0, ""},
	336920642:                                   {"option -sch needs a schema name", 0, ""},
	336920643:                                   {"option -sch got a too long schema name @1", 1, ""},
	336921576:                                   {"    -b      file blobs' statistics", 0, ""},
	336921577:                                   {"option -b needs a table.field name", 0, ""},
	336921578:                                   {"option -b got a too long table name @1", 1, ""},
	336921579:                                   {"option -b got a too long field name @1", 1, ""},
	336921580:                                   {"option -b accepts several table names only if used after <database>", 0, ""},
	336921581:                                   {"option -b field @1 of table @2 not found or not vaild", 2, ""},
	336921582:                                   {"option -b got an empty column name in parameter @1", 1, ""},
	336921583:                                   {"option -b got an empty table name in parameter @1", 1, ""},
	336921584:                                   {"option -b error while reading field @1 of table @2", 2, ""},
	336921585:                                   {"option -b cannot be entered more than once", 0, ""},
	336921586:                                   {"\nAnalyzing file blobs ...", 0, ""},
	336921587:                                   {"    -par    parallel workers number", 0, ""},
	336921588:                                   {"parallel workers parameter missing", 0, ""},
	336921589:                                   {"expected parallel workers, encountered @1", 1, ""},
	Isc_fbsvcmgr_bad_am:                         {"Wrong value for access mode", 0, "00000"},
	Isc_fbsvcmgr_bad_wm:                         {"Wrong value for write mode", 0, "00000"},
	Isc_fbsvcmgr_bad_rs:                         {"Wrong value for reserve space", 0, "00000"},
	Isc_fbsvcmgr_info_err:                       {"Unknown tag (@1) in info_svr_db_info block after isc_svc_query()", 1, "00000"},
	Isc_fbsvcmgr_query_err:                      {"Unknown tag (@1) in isc_svc_query() results", 1, "00000"},
	Isc_fbsvcmgr_switch_unknown:                 {"Unknown switch \"@1\"", 1, "00000"},
	336986119:                                   {"Service Manager Version", 0, ""},
	336986120:                                   {"Server version", 0, ""},
	336986121:                                   {"Server implementation", 0, ""},
	336986122:                                   {"Path to rdbserver.msg", 0, ""},
	336986123:                                   {"Server root", 0, ""},
	336986124:                                   {"Path to lock files", 0, ""},
	336986125:                                   {"Security database", 0, ""},
	336986126:                                   {"Databases", 0, ""},
	336986127:                                   {"   Database in use", 0, ""},
	336986128:                                   {"   Number of attachments", 0, ""},
	336986129:                                   {"   Number of databases", 0, ""},
	336986130:                                   {"Information truncated", 0, ""},
	336986131:                                   {"Usage: rdbsvcmgr manager-name switches...", 0, ""},
	336986132:                                   {"Manager-name should be service_mgr, may be prefixed with host name", 0, ""},
	336986133:                                   {"according to common rules (host:service_mgr, \\\\host\\service_mgr).", 0, ""},
	336986134:                                   {"Switches exactly match SPB tags, used in abbreviated form.", 0, ""},
	336986135:                                   {"Remove isc_, spb_ and svc_ parts of tag and you will get the switch.", 0, ""},
	336986136:                                   {"For example: isc_action_svc_backup is specified as action_backup,", 0, ""},
	336986137:                                   {"             isc_spb_dbname => dbname,", 0, ""},
	336986138:                                   {"             isc_info_svc_implementation => info_implementation,", 0, ""},
	336986139:                                   {"             isc_spb_prp_db_online => prp_db_online and so on.", 0, ""},
	336986140:                                   {"You may specify single action or multiple info items when calling rdbsvcmgr once.", 0, ""},
	336986141:                                   {"Full command line samples:", 0, ""},
	336986142:                                   {"rdbsvcmgr service_mgr user sysdba password masterkey action_db_stats dbname employee sts_hdr_pages", 0, ""},
	336986143:                                   {"  (will list header info in database employee on local machine)", 0, ""},
	336986144:                                   {"rdbsvcmgr yourserver:service_mgr user sysdba password masterkey info_server_version info_svr_db_info", 0, ""},
	336986145:                                   {"  (will show Red Database version and databases usage on yourserver)", 0, ""},
	336986146:                                   {"Transaction in limbo", 0, ""},
	336986147:                                   {"Multidatabase transaction in limbo", 0, ""},
	336986148:                                   {"Host Site", 0, ""},
	336986149:                                   {"Transaction", 0, ""},
	336986150:                                   {"has been prepared", 0, ""},
	336986151:                                   {"has been committed", 0, ""},
	336986152:                                   {"has been rolled back", 0, ""},
	336986153:                                   {"is not available", 0, ""},
	336986154:                                   {"Remote Site", 0, ""},
	336986155:                                   {"Database Path", 0, ""},
	336986156:                                   {"Automated recovery would commit this transaction", 0, ""},
	336986157:                                   {"Automated recovery would rollback this transaction", 0, ""},
	336986158:                                   {"No idea should it be commited or rolled back", 0, ""},
	Isc_fbsvcmgr_bad_sm:                         {"Wrong value for shutdown mode", 0, "00000"},
	Isc_fbsvcmgr_fp_open:                        {"could not open file @1", 1, "00000"},
	Isc_fbsvcmgr_fp_read:                        {"could not read file @1", 1, "00000"},
	Isc_fbsvcmgr_fp_empty:                       {"empty file @1", 1, "00000"},
	336986163:                                   {"Red Database Services Manager version @1", 1, ""},
	Isc_fbsvcmgr_bad_arg:                        {"Invalid or missing parameter for switch @1", 1, "00000"},
	336986165:                                   {"To get full list of known services run with -? switch", 0, ""},
	336986166:                                   {"Attaching to services manager:", 0, ""},
	336986167:                                   {"Information requests:", 0, ""},
	336986168:                                   {"Actions:", 0, ""},
	336986169:                                   {"Server capabilities:", 0, ""},
	Isc_fbsvcmgr_info_limbo:                     {"Unknown tag (@1) in isc_info_svc_limbo_trans block after isc_svc_query()", 1, "00000"},
	Isc_fbsvcmgr_limbo_state:                    {"Unknown tag (@1) in isc_spb_tra_state block after isc_svc_query()", 1, "00000"},
	Isc_fbsvcmgr_limbo_advise:                   {"Unknown tag (@1) in isc_spb_tra_advise block after isc_svc_query()", 1, "00000"},
	Isc_fbsvcmgr_bad_rm:                         {"Wrong value for replica mode", 0, "00000"},
	Isc_utl_trusted_switch:                      {"Switches trusted_user and trusted_role are not supported from command line", 0, "00000"},
	337117185:                                   {"ERROR: ", 0, ""},
	337117186:                                   {"Physical Backup Manager    Copyright (C) 2004 Firebird/Red Database development team", 0, ""},
	337117187:                                   {"  Original idea is of Sean Leyne <sean@@broadviewsoftware.com>", 0, ""},
	337117188:                                   {"  Designed and implemented by Nickolay Samofatov <skidder@@bssys.com>", 0, ""},
	337117189:                                   {"  This work was funded through a grant from BroadView Software, Inc.\\n", 0, ""},
	337117190:                                   {"Usage: nbackup <options>", 0, ""},
	337117191:                                   {"exclusive options are:", 0, ""},
	337117192:                                   {"  -L(OCK) <database>                     Lock database for filesystem copy", 0, ""},
	337117193:                                   {"  -UN(LOCK) <database>                   Unlock previously locked database", 0, ""},
	337117194:                                   {"  -F(IXUP) <database>                    Fixup database after filesystem copy", 0, ""},
	337117195:                                   {"  -B(ACKUP) <level>|<GUID> <db> [<file>] Create incremental backup", 0, ""},
	337117196:                                   {"  -R(ESTORE) <db> [<file0> [<file1>...]] Restore incremental backup", 0, ""},
	337117197:                                   {"  -U(SER) <user>                         User name", 0, ""},
	337117198:                                   {"  -P(ASSWORD) <password>                 Password", 0, ""},
	337117199:                                   {"  -FETCH_PASSWORD <file>                 Fetch password from file", 0, ""},
	337117200:                                   {"  -NOD(BTRIGGERS)                        Do not run database triggers", 0, ""},
	337117201:                                   {"  -S(IZE)                                Print database size in pages after lock", 0, ""},
	337117202:                                   {"  -Z                                     Print program version", 0, ""},
	337117203:                                   {"Notes:", 0, ""},
	337117204:                                   {"  <database> may specify database alias.", 0, ""},
	337117205:                                   {"  Incremental backups of multi-file databases are not supported yet.", 0, ""},
	337117206:                                   {"  \"stdout\" may be used as a value of <filename> for -B option.", 0, ""},
	337117207:                                   {"PROBLEM ON \"@1\".", 1, ""},
	337117208:                                   {"general options are:", 0, ""},
	337117209:                                   {"switches can be abbreviated to the unparenthesized characters", 0, ""},
	337117210:                                   {"  Option -S(IZE) only is valid together with -L(OCK).", 0, ""},
	337117211:                                   {"  For historical reasons, -N is equivalent to -UN(LOCK)", 0, ""},
	337117212:                                   {"  and -T is equivalent to -NOD(BTRIGGERS).", 0, ""},
	Isc_nbackup_missing_param:                   {"Missing parameter for switch @1", 1, "00000"},
	Isc_nbackup_allowed_switches:                {"Only one of -LOCK, -UNLOCK, -FIXUP, -BACKUP or -RESTORE should be specified", 0, "00000"},
	Isc_nbackup_unknown_param:                   {"Unrecognized parameter @1", 1, "00000"},
	Isc_nbackup_unknown_switch:                  {"Unknown switch @1", 1, "00000"},
	Isc_nbackup_nofetchpw_svc:                   {"Fetch password can't be used in service mode", 0, "00000"},
	Isc_nbackup_pwfile_error:                    {"Error working with password file \"@1\"", 1, "00000"},
	Isc_nbackup_size_with_lock:                  {"Switch -SIZE can be used only with -LOCK", 0, "00000"},
	Isc_nbackup_no_switch:                       {"None of -LOCK, -UNLOCK, -FIXUP, -BACKUP or -RESTORE specified", 0, "00000"},
	337117221:                                   {"Failure: ", 0, ""},
	337117222:                                   {"Enter name of the backup file of level @1 (\".\" - do not restore further):", 1, ""},
	Isc_nbackup_err_read:                        {"IO error reading file: @1", 1, "00000"},
	Isc_nbackup_err_write:                       {"IO error writing file: @1", 1, "00000"},
	Isc_nbackup_err_seek:                        {"IO error seeking file: @1", 1, "00000"},
	Isc_nbackup_err_opendb:                      {"Error opening database file: @1", 1, "00000"},
	Isc_nbackup_err_fadvice:                     {"Error in posix_fadvise(@1) for database @2", 2, "00000"},
	Isc_nbackup_err_createdb:                    {"Error creating database file: @1", 1, "00000"},
	Isc_nbackup_err_openbk:                      {"Error opening backup file: @1", 1, "00000"},
	Isc_nbackup_err_createbk:                    {"Error creating backup file: @1", 1, "00000"},
	Isc_nbackup_err_eofdb:                       {"Unexpected end of database file @1", 1, "00000"},
	Isc_nbackup_fixup_wrongstate:                {"Database @1 is not in state (@2) to be safely fixed up", 2, "00000"},
	Isc_nbackup_err_db:                          {"Database error", 0, "00000"},
	Isc_nbackup_userpw_toolong:                  {"Username or password is too long", 0, "00000"},
	Isc_nbackup_lostrec_db:                      {"Cannot find record for database \"@1\" backup level @2 in the backup history", 2, "00000"},
	Isc_nbackup_lostguid_db:                     {"Internal error. History query returned null SCN or GUID", 0, "00000"},
	Isc_nbackup_err_eofhdrdb:                    {"Unexpected end of file when reading header of database file \"@1\" (stage @2)", 2, "00000"},
	Isc_nbackup_db_notlock:                      {"Internal error. Database file is not locked. Flags are @1", 1, "00000"},
	Isc_nbackup_lostguid_bk:                     {"Internal error. Cannot get backup guid clumplet", 0, "00000"},
	Isc_nbackup_page_changed:                    {"Internal error. Database page @1 had been changed during backup (page SCN=@2, backup SCN=@3)", 3, "00000"},
	Isc_nbackup_dbsize_inconsistent:             {"Database file size is not a multiple of page size", 0, "00000"},
	Isc_nbackup_failed_lzbk:                     {"Level 0 backup is not restored", 0, "00000"},
	Isc_nbackup_err_eofhdrbk:                    {"Unexpected end of file when reading header of backup file: @1", 1, "00000"},
	Isc_nbackup_invalid_incbk:                   {"Invalid incremental backup file: @1", 1, "00000"},
	Isc_nbackup_unsupvers_incbk:                 {"Unsupported version @1 of incremental backup file: @2", 2, "00000"},
	Isc_nbackup_invlevel_incbk:                  {"Invalid level @1 of incremental backup file: @2, expected @3", 3, "00000"},
	Isc_nbackup_wrong_orderbk:                   {"Wrong order of backup files or invalid incremental backup file detected, file: @1", 1, "00000"},
	Isc_nbackup_err_eofbk:                       {"Unexpected end of backup file: @1", 1, "00000"},
	Isc_nbackup_err_copy:                        {"Error creating database file: @1 via copying from: @2", 2, "00000"},
	Isc_nbackup_err_eofhdr_restdb:               {"Unexpected end of file when reading header of restored database file (stage @1)", 1, "00000"},
	Isc_nbackup_lostguid_l0bk:                   {"Cannot get backup guid clumplet from L0 backup", 0, "00000"},
	337117252:                                   {"Physical Backup Manager version @1", 1, ""},
	337117253:                                   {"Enter name of the backup file of level @1 (\".\" - do not restore further):", 1, ""},
	337117254:                                   {"  -D(IRECT) <ON | OFF>                   Use or not direct I/O when backing up database", 0, ""},
	Isc_nbackup_switchd_parameter:               {"Wrong parameter @1 for switch -D, need ON or OFF", 1, "00000"},
	337117256:                                   {"special options are:", 0, ""},
	Isc_nbackup_user_stop:                       {"Terminated due to user request", 0, "08006"},
	337117258:                                   {"  -DE(COMPRESS) <command>                Command to extract archives during restore", 0, ""},
	Isc_nbackup_deco_parse:                      {"Too complex decompress command (> @1 arguments)", 1, "54023"},
	337117260:                                   {"  -RO(LE) <role>                         SQL role name", 0, ""},
	Isc_nbackup_lostrec_guid_db:                 {"Cannot find record for database \"@1\" backup GUID @2 in the backup history", 2, "00000"},
	337117262:                                   {"  -I(NPLACE)                             Restore incremental backup(s) to existing database", 0, ""},
	337117263:                                   {"  -INPLACE option could corrupt the database that has changed since previous restore.", 0, ""},
	337117264:                                   {"  -SEQ(UENCE)                            Preserve original replication sequence", 0, ""},
	Isc_nbackup_seq_misuse:                      {"Switch -SEQ(UENCE) can be used only with -FIXUP or -RESTORE", 0, "00000"},
	337117266:                                   {"  -CLEAN_HIST(ORY)                       Clean old records from backup history", 0, ""},
	337117267:                                   {"  -K(EEP) <N> <(R)OWS | (D)AYS>          How many recent rows (or days back from today) should be kept in the history", 0, ""},
	Isc_nbackup_wrong_param:                     {"Wrong parameter value for switch @1", 1, "00000"},
	Isc_nbackup_clean_hist_misuse:               {"Switch -CLEAN_HISTORY can be used only with -BACKUP", 0, "00000"},
	Isc_nbackup_clean_hist_missed:               {"-KEEP can be used only with -CLEAN_HISTORY", 0, "00000"},
	Isc_nbackup_keep_hist_missed:                {"-KEEP is required with -CLEAN_HISTORY", 0, "00000"},
	Isc_nbackup_second_keep_switch:              {"-KEEP can be used one time only", 0, "00000"},
	337118184:                                   {"  \"stdin\" may be used for -R option.", 0, ""},
	337118185:                                   {"Cannot find record for database \"@1\" backup GUID @2 in the backup history", 2, ""},
	337118186:                                   {"  -I                                     Restore incremental backup(s) to existing database", 0, ""},
	337118187:                                   {"  -I option could corrupt the database that has changed since previous restore", 0, ""},
	337182721:                                   {"Red Database Trace Manager version @1", 1, ""},
	337182722:                                   {"ERROR: ", 0, ""},
	337182723:                                   {"Red Database Trace Manager.", 0, ""},
	337182724:                                   {"Usage: rdbtracemgr <action> [<parameters>]", 0, ""},
	337182725:                                   {"Actions:", 0, ""},
	337182726:                                   {"  -STA[RT]                              Start trace session", 0, ""},
	337182727:                                   {"  -STO[P]                               Stop trace session", 0, ""},
	337182728:                                   {"  -SU[SPEND]                            Suspend trace session", 0, ""},
	337182729:                                   {"  -R[ESUME]                             Resume trace session", 0, ""},
	337182730:                                   {"  -L[IST]                               List existing trace sessions", 0, ""},
	337182731:                                   {"  -Z                                    Show program version", 0, ""},
	337182732:                                   {"Action parameters:", 0, ""},
	337182733:                                   {"  -N[AME]    <string>                   Session name", 0, ""},
	337182734:                                   {"  -I[D]      <number>                   Session ID", 0, ""},
	337182735:                                   {"  -C[ONFIG]  <string>                   Trace configuration file name", 0, ""},
	337182736:                                   {"Connection parameters:", 0, ""},
	337182737:                                   {"  -SE[RVICE]  <string>                  Service name", 0, ""},
	337182738:                                   {"  -U[SER]     <string>                  User name", 0, ""},
	337182739:                                   {"  -P[ASSWORD] <string>                  Password", 0, ""},
	337182740:                                   {"  -FE[TCH]    <string>                  Fetch password from file", 0, ""},
	337182741:                                   {"  -T[RUSTED]                            Force trusted authentication", 0, ""},
	337182742:                                   {"Examples:", 0, ""},
	337182743:                                   {"  rdbtracemgr -SE remote_host:service_mgr -USER SYSDBA -PASS masterkey -LIST", 0, ""},
	337182744:                                   {"  rdbtracemgr -SE service_mgr -START -NAME my_trace -CONFIG my_cfg.txt", 0, ""},
	337182745:                                   {"  rdbtracemgr -SE service_mgr -SUSPEND -ID 2", 0, ""},
	337182746:                                   {"  rdbtracemgr -SE service_mgr -RESUME -ID 2", 0, ""},
	337182747:                                   {"  rdbtracemgr -SE service_mgr -STOP -ID 4", 0, ""},
	337182748:                                   {"Notes:", 0, ""},
	337182749:                                   {"  Press CTRL+C to stop interactive trace session", 0, ""},
	Isc_trace_conflict_acts:                     {"conflicting actions \"@1\" and \"@2\" found", 2, "00000"},
	Isc_trace_act_notfound:                      {"action switch not found", 0, "00000"},
	Isc_trace_switch_once:                       {"switch \"@1\" must be set only once", 1, "00000"},
	Isc_trace_param_val_miss:                    {"value for switch \"@1\" is missing", 1, "00000"},
	Isc_trace_param_invalid:                     {"invalid value (\"@1\") for switch \"@2\"", 2, "00000"},
	Isc_trace_switch_unknown:                    {"unknown switch \"@1\" encountered", 1, "00000"},
	Isc_trace_switch_svc_only:                   {"switch \"@1\" can be used by service only", 1, "00000"},
	Isc_trace_switch_user_only:                  {"switch \"@1\" can be used by interactive user only", 1, "00000"},
	Isc_trace_switch_param_miss:                 {"mandatory parameter \"@1\" for switch \"@2\" is missing", 2, "00000"},
	Isc_trace_param_act_notcompat:               {"parameter \"@1\" is incompatible with action \"@2\"", 2, "00000"},
	Isc_trace_mandatory_switch_miss:             {"mandatory switch \"@1\" is missing", 1, "00000"},
	337182761:                                   {"  -P[LUGINS]  <string list>             Plugins list for use with trace session; valid list separators: <tab>, <comma>, <semicolon>", 0, ""},
	337182762:                                   {"  fbtracemgr -SE service_mgr -START -NAME my_trace -CONFIG my_cfg.txt -PLUGINS fbtrace,custom_plugin", 0, ""},
	337182820:                                   {"  -F[ORMAT]   <number>                  Trace output format: 0 for text, 1 for binary, and -1 for legacy behavior", 0, ""},
	337182821:                                   {"  -O[UTPUT]   <string>                  Output file", 0, ""},
	337182822:                                   {"  -REP[LACE]                            Overwrite the output file", 0, ""},
	337182830:                                   {"  rdbtracemgr -SE service_mgr -START -NAME my_trace -CONFIG my_cfg.txt -format 0 -output /tmp/log.txt", 0, ""},
	337182831:                                   {"  rdbtracemgr -SE service_mgr -START -NAME my_trace -CONFIG my_cfg.txt -format 1 -output /tmp/log.bin", 0, ""},
	337182840:                                   {"  For format 1, you must specify an output file", 0, ""},
	337510401:                                   {"Usage:", 0, ""},
	337510402:                                   {" -D[ATABASE] <database> : master database name", 0, ""},
	337510403:                                   {" -R[EPLICA] <database>  : replica database name (<user>:<pwd>@@database for simple mode)", 0, ""},
	337510404:                                   {" -U[SER]     <string>   : user name", 0, ""},
	337510405:                                   {" -P[ASSWORD] <string>   : password", 0, ""},
	337510406:                                   {" -C[ONFIG] <config>     : configuration file", 0, ""},
	337510407:                                   {" -M[ETADATA]            : compare only metadata", 0, ""},
	337510408:                                   {" -S[YNCHRONIZE]         : wait until master and slave synchronize", 0, ""},
	337510409:                                   {" -T[IMEOUT]             : timeout (in seconds) to wait for synchronization", 0, ""},
	337510410:                                   {" -N[odbtriggersoff]     : do run database triggers", 0, ""},
	337510411:                                   {" -SI[MPLE]              : compare non-replicated databases", 0, ""},
	337510412:                                   {" -V[ERBOSE]             : verbose output", 0, ""},
	337510413:                                   {" -Z                     : report version", 0, ""},
	337510414:                                   {" -?                     : help", 0, ""},
	Isc_diff_incmp_sw:                           {"incompatible switch combination", 0, "00000"},
	Isc_diff_unicode_init_error:                 {"Cannot initialize UNICODE collation", 0, "00000"},
	Isc_diff_replconf_section_not_found:         {"Replication config: @1 section not found", 1, "00000"},
	Isc_DiffErr_slave_name:                      {"ERROR: The transferred slave name does not match the database_replication parameter in the config", 0, ""},
	Isc_DiffErr_records_number:                  {"ERROR: Number of records does not match in table @1", 1, ""},
	Isc_DiffErr_records:                         {"ERROR: Records in table @1 do not match", 1, ""},
	Isc_DiffErr_tables_number:                   {"ERROR: Number of tables does not match", 0, ""},
	Isc_DiffErr_tables_list:                     {"ERROR: List of tables does not match", 0, ""},
	Isc_DiffErr_primary_keys:                    {"ERROR: Primary keys for table @1 do not match", 1, ""},
	Isc_DiffErr_blobs:                           {"ERROR: Blobs in table @1 do not match", 1, ""},
	Isc_DiffErr_remote_master:                   {"ERROR: Only local master database can be validated", 0, ""},
	Isc_DiffErr_sync_timeout:                    {"ERROR: Master and slave are not in sync after the specified timeout", 0, ""},
	Isc_DiffMgs_waiting:                         {"Waiting for the replica to catch up the master database... ", 0, ""},
	Isc_DiffMgs_checking:                        {"Checking table @1... ", 1, ""},
	Isc_DiffMgs_failed:                          {"FAILED\n\t", 0, ""},
	Isc_DiffMgs_ok_empty:                        {"OK (empty)(hash)", 0, ""},
	Isc_DiffMgs_on_compared:                     {"OK (@1 records compared)@2", 2, ""},
	Isc_DiffMgs_summary_no_diff:                 {"Summary: no differences found", 0, ""},
	Isc_DiffMgs_summary_diff:                    {"Summary: @1 table(s) have different data", 1, ""},
	Isc_DiffMgs_summary_error:                   {"Summary: error during processing", 0, ""},
	Isc_DiffMgs_succeeded:                       {"succeeded", 0, ""},
	Isc_DiffMsg_version:                         {"Replica validation utility, version @1", 1, ""},
	Isc_diff_param_val_miss:                     {"Value for switch \"@1\" is missing", 1, "00000"},
	337510438:                                   {" -F[ULL]                : find all differences", 0, ""},
	Isc_DiffMsg_table_name:                      {"ERROR: Table @1", 1, ""},
	Isc_DiffErr_index_name:                      {"ERROR: Index name for table @1 does not match", 1, ""},
	Isc_DiffErr_attributes:                      {"ERROR: Attributes in table @1 do not match", 1, ""},
	Isc_DiffErr_field_count:                     {"ERROR: Field count for table @1 does not match", 1, ""},
	Isc_DiffErr_field_names:                     {"ERROR: Field names for table @1 does not match", 1, ""},
	Isc_DiffErrFull_not_found:                   {"Record ID: @1, not found", 1, ""},
	Isc_DiffErrFull_attributes:                  {"Record ID: @1, Attributes do not match (field: @2, type: @3)", 3, ""},
	Isc_DiffErrFull_blobs:                       {"Record ID: @1, Blobs do not match (field: @2, type: @3)", 3, ""},
	Isc_DiffErrFull_records:                     {"Record ID: @1, Records data do not match (field: @2, type: @3)", 3, ""},
	Isc_DiffErrFull_master_has_more_records:     {"Master has @1 additional record(s)", 1, ""},
	Isc_DiffErrFull_slave_has_more_records:      {"Slave has @1 additional record(s)", 1, ""},
	Isc_DiffErrFull_table_not_found:             {"ERROR: Slave does not have %s table", 0, ""},
	Isc_DiffErrFull_master_has_more_tables:      {"\nMaster has @1 additional table(s)", 1, ""},
	Isc_DiffErrFull_slave_has_more_tables:       {"\nSlave has @1 additional table(s)", 1, ""},
	Isc_DiffErrFull_records_value:               {"Record ID: @1, data do not match in @2 field (@3 vs @4)", 4, ""},
	337510454:                                   {" -VA[LUES]              : print mismatched values instead of types", 0, ""},
}
